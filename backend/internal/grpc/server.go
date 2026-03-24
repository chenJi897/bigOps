package grpc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"gorm.io/gorm"

	pb "github.com/bigops/platform/proto/gen/agent"

	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/pkg/database"
	"github.com/bigops/platform/internal/pkg/logger"
	"github.com/bigops/platform/internal/repository"
)

// Server implements the AgentServiceServer interface from generated gRPC code.
type Server struct {
	pb.UnimplementedAgentServiceServer
	agentRepo *repository.AgentRepository
	execRepo  *repository.TaskExecutionRepository
}

// Heartbeat handles bidirectional heartbeat streams from agents.
func (s *Server) Heartbeat(stream grpc.BidiStreamingServer[pb.HeartbeatRequest, pb.HeartbeatResponse]) error {
	// First message identifies the agent
	firstMsg, err := stream.Recv()
	if err != nil {
		return err
	}

	agentID := firstMsg.GetAgentId()
	if agentID == "" {
		return fmt.Errorf("agent_id is required in first heartbeat message")
	}

	ctx, cancel := context.WithCancel(stream.Context())
	defer cancel()

	mgr := GetAgentManager()
	hostname := firstMsg.GetHostname()
	ip := firstMsg.GetIp()

	// Register agent, upsert full info to DB
	now := model.LocalTime(time.Now())
	labelsJSON := "{}"
	if labels := firstMsg.GetLabels(); len(labels) > 0 {
		if data, err := json.Marshal(labels); err == nil {
			labelsJSON = string(data)
		}
	}
	_ = s.agentRepo.Upsert(&model.AgentInfo{
		AgentID:       agentID,
		Hostname:      hostname,
		IP:            ip,
		Version:       firstMsg.GetVersion(),
		OS:            firstMsg.GetOs(),
		CPUCount:      int(firstMsg.GetCpuCount()),
		MemoryTotal:   firstMsg.GetMemoryTotal(),
		Labels:        labelsJSON,
		Status:        "online",
		LastHeartbeat: &now,
	})

	mgr.RegisterAgent(agentID, stream, cancel, hostname, ip)
	defer mgr.UnregisterAgent(agentID)

	// Send first ack
	_ = stream.Send(&pb.HeartbeatResponse{
		Accepted:   true,
		ServerTime: time.Now().Unix(),
	})

	// Loop: receive heartbeats, update last_heartbeat
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		msg, err := stream.Recv()
		if err == io.EOF {
			logger.Info("Agent heartbeat stream ended", zap.String("agent_id", agentID))
			return nil
		}
		if err != nil {
			logger.Warn("Agent heartbeat stream error", zap.String("agent_id", agentID), zap.Error(err))
			return err
		}

		// Update heartbeat timestamp in DB
		hbTime := model.LocalTime(time.Now())
		_ = s.agentRepo.Upsert(&model.AgentInfo{
			AgentID:       agentID,
			Hostname:      msg.GetHostname(),
			IP:            msg.GetIp(),
			Version:       msg.GetVersion(),
			OS:            msg.GetOs(),
			Status:        "online",
			LastHeartbeat: &hbTime,
		})

		// Send ack (no task in normal heartbeat responses)
		_ = stream.Send(&pb.HeartbeatResponse{
			Accepted:   true,
			ServerTime: time.Now().Unix(),
		})
	}
}

// ReportOutput receives streaming execution output from agents.
func (s *Server) ReportOutput(stream grpc.ClientStreamingServer[pb.ExecuteResponse, pb.ExecuteAck]) error {
	mgr := GetAgentManager()

	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			return stream.SendAndClose(&pb.ExecuteAck{Received: true})
		}
		if err != nil {
			logger.Warn("ReportOutput stream error", zap.Error(err))
			return err
		}

		hostResultID := msg.GetHostResultId()
		phase := msg.GetPhase()
		outputLine := msg.GetOutputLine()
		isStderr := msg.GetIsStderr()

		// Get host result to find execution context
		hr, err := s.execRepo.GetHostResult(hostResultID)
		if err != nil {
			logger.Warn("Host result not found", zap.Int64("host_result_id", hostResultID), zap.Error(err))
			continue
		}

		switch phase {
		case "running":
			// Mark as running if still pending
			updates := map[string]interface{}{}
			if hr.Status == "pending" {
				updates["status"] = "running"
				now := model.LocalTime(time.Now())
				updates["started_at"] = &now
			}
			// Atomic append output using CONCAT to avoid race conditions
			appendLine := outputLine + "\n"
			if isStderr {
				updates["stderr"] = gorm.Expr("CONCAT(COALESCE(stderr,''), ?)", appendLine)
			} else {
				updates["stdout"] = gorm.Expr("CONCAT(COALESCE(stdout,''), ?)", appendLine)
			}
			database.GetDB().Model(&model.TaskHostResult{}).Where("id = ?", hr.ID).Updates(updates)

			// Publish to WebSocket subscribers
			mgr.PublishLog(hr.ExecutionID, &LogLine{
				HostResultID: hostResultID,
				HostIP:       hr.HostIP,
				Line:         outputLine,
				IsStderr:     isStderr,
				Phase:        phase,
				Timestamp:    msg.GetTimestamp(),
			})

		case "finished", "error":
			// Final update
			now := model.LocalTime(time.Now())
			finalUpdates := map[string]interface{}{
				"finished_at": &now,
				"exit_code":   int(msg.GetExitCode()),
			}
			if phase == "finished" && msg.GetExitCode() == 0 {
				finalUpdates["status"] = "success"
			} else {
				finalUpdates["status"] = "failed"
			}
			// Append any remaining output
			if outputLine != "" {
				appendLine := outputLine + "\n"
				if isStderr {
					finalUpdates["stderr"] = gorm.Expr("CONCAT(COALESCE(stderr,''), ?)", appendLine)
				} else {
					finalUpdates["stdout"] = gorm.Expr("CONCAT(COALESCE(stdout,''), ?)", appendLine)
				}
			}
			database.GetDB().Model(&model.TaskHostResult{}).Where("id = ?", hr.ID).Updates(finalUpdates)

			// Publish finish event to WebSocket
			mgr.PublishLog(hr.ExecutionID, &LogLine{
				HostResultID: hostResultID,
				HostIP:       hr.HostIP,
				Line:         outputLine,
				IsStderr:     isStderr,
				Phase:        phase,
				ExitCode:     msg.GetExitCode(),
				Timestamp:    msg.GetTimestamp(),
			})

			// Check if all host results for this execution are done
			s.checkExecutionCompletion(hr.ExecutionID)
		}
	}
}

// FileTransfer handles file uploads from agents (stub for now).
func (s *Server) FileTransfer(stream grpc.ClientStreamingServer[pb.FileChunk, pb.TransferResult]) error {
	var totalBytes int64
	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			return stream.SendAndClose(&pb.TransferResult{
				Success:       true,
				Message:       "transfer complete",
				BytesReceived: totalBytes,
			})
		}
		if err != nil {
			return err
		}
		totalBytes += int64(len(chunk.GetChunkData()))
	}
}

// checkExecutionCompletion checks if all host results are done and updates execution status.
func (s *Server) checkExecutionCompletion(executionID int64) {
	results, err := s.execRepo.GetHostResultsByExecutionID(executionID)
	if err != nil {
		return
	}

	allDone := true
	successCount := 0
	failCount := 0
	for _, r := range results {
		switch r.Status {
		case "success":
			successCount++
		case "failed", "timeout":
			failCount++
		default:
			allDone = false
		}
	}

	if !allDone {
		return
	}

	exec, err := s.execRepo.GetByID(executionID)
	if err != nil {
		return
	}

	now := model.LocalTime(time.Now())
	exec.FinishedAt = &now
	exec.SuccessCount = successCount
	exec.FailCount = failCount

	if failCount == 0 {
		exec.Status = "success"
	} else if successCount == 0 {
		exec.Status = "failed"
	} else {
		exec.Status = "partial_fail"
	}

	_ = s.execRepo.Update(exec)

	logger.Info("Execution completed",
		zap.Int64("execution_id", executionID),
		zap.String("status", exec.Status),
		zap.Int("success", successCount),
		zap.Int("fail", failCount),
	)
}

// StartGRPCServer creates and starts the gRPC server on the given port.
func StartGRPCServer(port int) (*grpc.Server, error) {
	agentRepo := repository.NewAgentRepository()
	execRepo := repository.NewTaskExecutionRepository()

	InitAgentManager(agentRepo)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, fmt.Errorf("failed to listen on port %d: %w", port, err)
	}

	srv := grpc.NewServer()
	pb.RegisterAgentServiceServer(srv, &Server{
		agentRepo: agentRepo,
		execRepo:  execRepo,
	})

	go func() {
		logger.Info("gRPC server listening", zap.Int("port", port))
		if err := srv.Serve(lis); err != nil {
			logger.Error("gRPC server error", zap.Error(err))
		}
	}()

	return srv, nil
}
