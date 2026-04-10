package handler

import (
	"strings"
	"time"

	"github.com/gorilla/websocket"

	agentgrpc "github.com/bigops/platform/internal/grpc"
)

// WSExecutionLogEvent WebSocket 推送的统一日志事件（P0 契约，与前端 TaskInstanceDetail 对齐）。
type WSExecutionLogEvent struct {
	ExecutionID  int64  `json:"execution_id"`
	Timestamp    int64  `json:"timestamp"`
	Content      string `json:"content"`
	IsStderr     bool   `json:"is_stderr"`
	Phase        string `json:"phase"`
	HostIP       string `json:"host_ip,omitempty"`
	HostResultID int64  `json:"host_result_id,omitempty"`
	ExitCode     int32  `json:"exit_code,omitempty"`
	DoneCount    int    `json:"done_count,omitempty"`
	TotalCount   int    `json:"total_count,omitempty"`
	SuccessCount int    `json:"success_count,omitempty"`
	FailCount    int    `json:"fail_count,omitempty"`
}

const wsReplayMaxLines = 5000

func wsEventFromLogLine(executionID int64, l *agentgrpc.LogLine) WSExecutionLogEvent {
	ts := l.Timestamp
	if ts == 0 {
		ts = time.Now().Unix()
	}
	return WSExecutionLogEvent{
		ExecutionID:  executionID,
		Timestamp:    ts,
		Content:      l.Line,
		IsStderr:     l.IsStderr,
		Phase:        l.Phase,
		HostIP:       l.HostIP,
		HostResultID: l.HostResultID,
		ExitCode:     l.ExitCode,
		DoneCount:    l.DoneCount,
		TotalCount:   l.TotalCount,
		SuccessCount: l.SuccessCount,
		FailCount:    l.FailCount,
	}
}

// writeExecutionLogReplay 将已落库的主机输出按行回放，便于断线重连后补齐历史。
// hostIPFilter 非空时仅回放该主机。
func (h *TaskHandler) writeExecutionLogReplay(conn *websocket.Conn, executionID int64, hostIPFilter string) error {
	exec, err := h.svc.GetExecution(executionID)
	if err != nil || exec == nil {
		return err
	}
	now := time.Now().Unix()
	emitted := 0

	flush := func(hostIP string, hrID int64, text string, isErr bool) error {
		text = strings.TrimSuffix(text, "\n")
		if text == "" {
			return nil
		}
		for _, line := range strings.Split(text, "\n") {
			if line == "" {
				continue
			}
			ev := WSExecutionLogEvent{
				ExecutionID:  executionID,
				Timestamp:    now,
				Content:      line,
				IsStderr:     isErr,
				Phase:        "replay",
				HostIP:       hostIP,
				HostResultID: hrID,
			}
			if err := conn.WriteJSON(ev); err != nil {
				return err
			}
			emitted++
			if emitted >= wsReplayMaxLines {
				return nil
			}
		}
		return nil
	}

	for i := range exec.HostResults {
		hr := &exec.HostResults[i]
		if hostIPFilter != "" && hr.HostIP != hostIPFilter {
			continue
		}
		if err := flush(hr.HostIP, hr.ID, hr.Stdout, false); err != nil {
			return err
		}
		if emitted >= wsReplayMaxLines {
			return nil
		}
		if err := flush(hr.HostIP, hr.ID, hr.Stderr, true); err != nil {
			return err
		}
		if emitted >= wsReplayMaxLines {
			return nil
		}
	}
	return nil
}
