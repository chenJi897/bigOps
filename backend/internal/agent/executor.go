package agent

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"sync"
	"time"

	pb "github.com/bigops/platform/proto/gen/agent"
)

type Executor struct{}

func NewExecutor() *Executor {
	return &Executor{}
}

func (e *Executor) Execute(ctx context.Context, task *pb.ExecuteRequest, stream pb.AgentService_ReportOutputClient) {
	timeout := time.Duration(task.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 60 * time.Second
	}

	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Build command based on script type
	cmd := e.buildCommand(execCtx, task)

	// Set up pipes
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		e.sendError(stream, task.HostResultId, fmt.Sprintf("stdout pipe error: %v", err))
		return
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		e.sendError(stream, task.HostResultId, fmt.Sprintf("stderr pipe error: %v", err))
		return
	}

	// Send running phase
	stream.Send(&pb.ExecuteResponse{
		HostResultId: task.HostResultId,
		Phase:        "running",
		Timestamp:    time.Now().Unix(),
	})

	if err := cmd.Start(); err != nil {
		e.sendError(stream, task.HostResultId, fmt.Sprintf("start error: %v", err))
		return
	}

	// Stream stdout and stderr concurrently
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		e.streamOutput(stream, task.HostResultId, stdout, false)
	}()

	go func() {
		defer wg.Done()
		e.streamOutput(stream, task.HostResultId, stderr, true)
	}()

	wg.Wait()

	// Wait for command to finish
	err = cmd.Wait()
	exitCode := 0
	phase := "finished"
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			phase = "error"
			exitCode = -1
		}
	}

	// Check if it was a timeout
	if execCtx.Err() == context.DeadlineExceeded {
		phase = "error"
		exitCode = -1
		stream.Send(&pb.ExecuteResponse{
			HostResultId: task.HostResultId,
			OutputLine:   "execution timed out",
			IsStderr:     true,
			Phase:        "error",
			ExitCode:     int32(exitCode),
			Timestamp:    time.Now().Unix(),
		})
		return
	}

	// Send finished
	stream.Send(&pb.ExecuteResponse{
		HostResultId: task.HostResultId,
		Phase:        phase,
		ExitCode:     int32(exitCode),
		Timestamp:    time.Now().Unix(),
	})

	log.Printf("Task host_result_id=%d finished with exit_code=%d", task.HostResultId, exitCode)
}

func (e *Executor) buildCommand(ctx context.Context, task *pb.ExecuteRequest) *exec.Cmd {
	var cmd *exec.Cmd
	switch task.ScriptType {
	case "python":
		cmd = exec.CommandContext(ctx, "python3", "-c", task.ScriptContent)
	case "powershell":
		cmd = exec.CommandContext(ctx, "powershell", "-Command", task.ScriptContent)
	default: // bash
		cmd = exec.CommandContext(ctx, "bash", "-c", task.ScriptContent)
	}

	if task.WorkDir != "" {
		cmd.Dir = task.WorkDir
	}

	// Set environment variables (inherit current env + add extras)
	if len(task.Env) > 0 {
		cmd.Env = os.Environ()
		for k, v := range task.Env {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
		}
	}

	return cmd
}

func (e *Executor) streamOutput(stream pb.AgentService_ReportOutputClient, hostResultID int64, reader io.Reader, isStderr bool) {
	scanner := bufio.NewScanner(reader)
	// Increase buffer size for long lines
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		stream.Send(&pb.ExecuteResponse{
			HostResultId: hostResultID,
			OutputLine:   line,
			IsStderr:     isStderr,
			Phase:        "running",
			Timestamp:    time.Now().Unix(),
		})
	}
}

func (e *Executor) sendError(stream pb.AgentService_ReportOutputClient, hostResultID int64, msg string) {
	log.Printf("Task error: host_result_id=%d: %s", hostResultID, msg)
	stream.Send(&pb.ExecuteResponse{
		HostResultId: hostResultID,
		OutputLine:   msg,
		IsStderr:     true,
		Phase:        "error",
		ExitCode:     -1,
		Timestamp:    time.Now().Unix(),
	})
}
