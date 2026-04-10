package agent

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/bigops/platform/internal/pkg/logger"
	"github.com/bigops/platform/internal/pkg/scriptguard"
	pb "github.com/bigops/platform/proto/gen/agent"
	"go.uber.org/zap"
)

type Executor struct{}

type reportOutputStream interface {
	Send(*pb.ExecuteResponse) error
	CloseAndRecv() (*pb.ExecuteAck, error)
}

func NewExecutor() *Executor {
	return &Executor{}
}

func (e *Executor) Execute(ctx context.Context, task *pb.ExecuteRequest, stream reportOutputStream) {
	timeout := time.Duration(task.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 60 * time.Second
	}

	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Validate script content before execution
	if err := scriptguard.Validate(task.ScriptContent, task.ScriptType); err != nil {
		e.sendError(stream, task.HostResultId, fmt.Sprintf("脚本安全检测未通过: %v", err))
		return
	}

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
	if err := stream.Send(&pb.ExecuteResponse{
		HostResultId: task.HostResultId,
		Phase:        "running",
		Timestamp:    time.Now().Unix(),
	}); err != nil {
		logger.Warn("Failed to send running phase",
			zap.Int64("host_result_id", task.HostResultId),
			zap.Error(err),
		)
		return
	}

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

	// Check if it was a timeout — kill entire process group
	if execCtx.Err() == context.DeadlineExceeded {
		phase = "error"
		exitCode = -1
		if runtime.GOOS == "linux" && cmd.Process != nil {
			_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		}
		if err := stream.Send(&pb.ExecuteResponse{
			HostResultId: task.HostResultId,
			OutputLine:   "execution timed out",
			IsStderr:     true,
			Phase:        "error",
			ExitCode:     int32(exitCode),
			Timestamp:    time.Now().Unix(),
		}); err != nil {
			logger.Warn("Failed to send timeout message",
				zap.Int64("host_result_id", task.HostResultId),
				zap.Error(err),
			)
		}
		return
	}

	// Send finished
	if err := stream.Send(&pb.ExecuteResponse{
		HostResultId: task.HostResultId,
		Phase:        phase,
		ExitCode:     int32(exitCode),
		Timestamp:    time.Now().Unix(),
	}); err != nil {
		logger.Warn("Failed to send finished message",
			zap.Int64("host_result_id", task.HostResultId),
			zap.Error(err),
		)
	}

	logger.Info("Task execution finished",
		zap.Int64("host_result_id", task.HostResultId),
		zap.Int("exit_code", exitCode),
		zap.String("phase", phase),
	)
}

func (e *Executor) buildCommand(ctx context.Context, task *pb.ExecuteRequest) *exec.Cmd {
	var cmd *exec.Cmd
	switch task.ScriptType {
	case "python":
		cmd = exec.CommandContext(ctx, "python3", "-c", task.ScriptContent)
	case "powershell":
		cmd = exec.CommandContext(ctx, "powershell", "-Command", task.ScriptContent)
	default: // bash
		// Inject ulimit resource limits: max procs 256, max file size 1GB, max open files 1024, no core dump
		guarded := "ulimit -u 256 -f 1048576 -n 1024 -c 0 2>/dev/null\n" + task.ScriptContent
		cmd = exec.CommandContext(ctx, "bash", "-c", guarded)
	}

	// Create new process group so we can kill the entire tree on timeout
	if runtime.GOOS == "linux" {
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
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

func (e *Executor) streamOutput(stream reportOutputStream, hostResultID int64, reader io.Reader, isStderr bool) {
	scanner := bufio.NewScanner(reader)
	// Increase buffer size for long lines
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if err := stream.Send(&pb.ExecuteResponse{
			HostResultId: hostResultID,
			OutputLine:   line,
			IsStderr:     isStderr,
			Phase:        "running",
			Timestamp:    time.Now().Unix(),
		}); err != nil {
			logger.Warn("Stream send failed",
				zap.Int64("host_result_id", hostResultID),
				zap.Error(err),
			)
			return
		}
	}
	if err := scanner.Err(); err != nil {
		logger.Warn("Scanner error",
			zap.Int64("host_result_id", hostResultID),
			zap.Error(err),
		)
	}
}

func (e *Executor) sendError(stream reportOutputStream, hostResultID int64, msg string) {
	logger.Error("Task execution error",
		zap.Int64("host_result_id", hostResultID),
		zap.String("error_msg", msg),
	)
	if err := stream.Send(&pb.ExecuteResponse{
		HostResultId: hostResultID,
		OutputLine:   msg,
		IsStderr:     true,
		Phase:        "error",
		ExitCode:     -1,
		Timestamp:    time.Now().Unix(),
	}); err != nil {
		logger.Warn("Failed to send error response",
			zap.Int64("host_result_id", hostResultID),
			zap.Error(err),
		)
	}
}
