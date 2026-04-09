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
	"google.golang.org/grpc/peer"
	"gorm.io/gorm"

	pb "github.com/bigops/platform/proto/gen/agent"

	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/pkg/database"
	"github.com/bigops/platform/internal/pkg/logger"
	"github.com/bigops/platform/internal/pkg/safego"
	"github.com/bigops/platform/internal/repository"
)

// Server implements the AgentServiceServer interface from generated gRPC code.
type Server struct {
	pb.UnimplementedAgentServiceServer
	agentRepo  *repository.AgentRepository
	execRepo   *repository.TaskExecutionRepository
	sampleRepo *repository.AgentMetricSampleRepository
	assetRepo  *repository.AssetRepository
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
	privateIP, publicIP := resolveAgentIPs(firstMsg, extractPeerIP(stream.Context()))
	displayIP := firstNonEmpty(publicIP, privateIP)

	// Register agent, upsert full info to DB
	now := model.LocalTime(time.Now())
	s.upsertHeartbeat(agentID, firstMsg, now, extractPeerIP(stream.Context()))

	mgr.RegisterAgent(agentID, stream, cancel, hostname, displayIP)
	defer mgr.UnregisterAgent(agentID)

	// Send first ack
	if err := stream.Send(&pb.HeartbeatResponse{
		Accepted:   true,
		ServerTime: time.Now().Unix(),
	}); err != nil {
		logger.Warn("Failed to send first heartbeat ack", zap.String("agent_id", agentID), zap.Error(err))
		return err
	}

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

		// Update heartbeat + resource metrics in DB
		hbTime := model.LocalTime(time.Now())
		s.upsertHeartbeat(agentID, msg, hbTime, extractPeerIP(stream.Context()))

		// Send ack (no task in normal heartbeat responses)
		if err := stream.Send(&pb.HeartbeatResponse{
			Accepted:   true,
			ServerTime: time.Now().Unix(),
		}); err != nil {
			logger.Warn("Failed to send heartbeat ack", zap.String("agent_id", agentID), zap.Error(err))
			return err
		}
	}
}

func (s *Server) upsertHeartbeat(agentID string, msg *pb.HeartbeatRequest, hbTime model.LocalTime, peerIP string) {
	if msg == nil {
		return
	}
	privateIP, publicIP := resolveAgentIPs(msg, peerIP)
	metricIP := firstNonEmpty(privateIP, publicIP)

	if err := s.agentRepo.Upsert(&model.AgentInfo{
		AgentID:        agentID,
		Hostname:       msg.GetHostname(),
		IP:             metricIP,
		PrivateIP:      privateIP,
		PublicIP:       publicIP,
		Version:        msg.GetVersion(),
		OS:             msg.GetOs(),
		Status:         "online",
		Labels:         labelsToJSON(msg.GetLabels()),
		CPUCount:       int(msg.GetCpuCount()),
		CPUUsagePct:    msg.GetCpuUsagePercent(),
		MemoryTotal:    msg.GetMemoryTotal(),
		MemoryUsed:     msg.GetMemoryUsed(),
		MemoryUsagePct: msg.GetMemoryUsagePercent(),
		DiskTotal:      msg.GetDiskTotal(),
		DiskUsed:       msg.GetDiskUsed(),
		DiskUsagePct:   msg.GetDiskUsagePercent(),
		LastHeartbeat:  &hbTime,
	}); err != nil {
		logger.Warn("upsert agent heartbeat failed", zap.String("agent_id", agentID), zap.Error(err))
	}

	samples := []model.AgentMetricSample{
		{
			AgentID:     agentID,
			Hostname:    msg.GetHostname(),
			IP:          metricIP,
			MetricType:  "cpu_usage",
			MetricValue: msg.GetCpuUsagePercent(),
			Unit:        "%",
			CollectedAt: hbTime,
		},
		{
			AgentID:     agentID,
			Hostname:    msg.GetHostname(),
			IP:          metricIP,
			MetricType:  "memory_usage",
			MetricValue: msg.GetMemoryUsagePercent(),
			Unit:        "%",
			CollectedAt: hbTime,
		},
		{
			AgentID:     agentID,
			Hostname:    msg.GetHostname(),
			IP:          metricIP,
			MetricType:  "disk_usage",
			MetricValue: msg.GetDiskUsagePercent(),
			Unit:        "%",
			CollectedAt: hbTime,
		},
	}
	if s.sampleRepo == nil {
		s.sampleRepo = repository.NewAgentMetricSampleRepository()
	}
	if err := s.sampleRepo.CreateBatch(samples); err != nil {
		logger.Warn("create metric samples failed", zap.String("agent_id", agentID), zap.Error(err))
	}

	s.ensureAgentAsset(msg, peerIP)
}

func (s *Server) ensureAgentAsset(msg *pb.HeartbeatRequest, peerIP string) {
	if msg == nil || msg.GetHostname() == "" {
		return
	}
	if s.assetRepo == nil {
		s.assetRepo = repository.NewAssetRepository()
	}
	privateIP, publicIP := resolveAgentIPs(msg, peerIP)
	primaryIP := firstNonEmpty(publicIP, privateIP)
	if primaryIP == "" {
		return
	}

	existing, err := s.assetRepo.GetByAnyIP(primaryIP)
	if err != nil && privateIP != "" && privateIP != primaryIP {
		existing, err = s.assetRepo.GetByAnyIP(privateIP)
	}
	if err == nil {
		updated := false
		if existing.Hostname == "" && msg.GetHostname() != "" {
			existing.Hostname = msg.GetHostname()
			updated = true
		}
		if publicIP != "" && existing.IP != publicIP {
			existing.IP = publicIP
			updated = true
		}
		if privateIP != "" && existing.InnerIP != privateIP {
			existing.InnerIP = privateIP
			updated = true
		}
		if existing.IP == "" {
			existing.IP = primaryIP
			updated = true
		}
		if existing.OS == "" && msg.GetOs() != "" {
			existing.OS = msg.GetOs()
			updated = true
		}
		if existing.Status == "" {
			existing.Status = "online"
			updated = true
		}
		if existing.Source == "" {
			existing.Source = "agent"
			updated = true
		}
		if updated {
			_ = s.assetRepo.Update(existing)
		}
		return
	}

	asset := &model.Asset{
		Hostname:  msg.GetHostname(),
		IP:        primaryIP,
		InnerIP:   privateIP,
		OS:        msg.GetOs(),
		Status:    "online",
		AssetType: "server",
		Source:    "agent",
		CPUCores:  int(msg.GetCpuCount()),
		MemoryMB:  int(msg.GetMemoryTotal() / 1024 / 1024),
		DiskGB:    int(msg.GetDiskTotal() / 1024 / 1024 / 1024),
		Remark:    "由 Agent 心跳自动创建",
	}
	if err := s.assetRepo.Create(asset); err != nil {
		logger.Warn("auto create asset from agent heartbeat failed", zap.String("hostname", asset.Hostname), zap.String("ip", asset.IP), zap.Error(err))
	}
}

func extractPeerIP(ctx context.Context) string {
	info, ok := peer.FromContext(ctx)
	if !ok || info == nil || info.Addr == nil {
		return ""
	}
	host, _, err := net.SplitHostPort(info.Addr.String())
	if err == nil {
		return host
	}
	return info.Addr.String()
}

func resolveAgentIPs(msg *pb.HeartbeatRequest, peerIP string) (privateIP string, publicIP string) {
	privateIP = firstNonEmpty(msg.GetPrivateIp(), msg.GetIp())
	publicIP = msg.GetPublicIp()
	if publicIP == "" && isPublicIP(peerIP) && peerIP != privateIP {
		publicIP = peerIP
	}
	return privateIP, publicIP
}

func firstNonEmpty(items ...string) string {
	for _, item := range items {
		if item != "" {
			return item
		}
	}
	return ""
}

func isPublicIP(raw string) bool {
	ip := net.ParseIP(raw)
	if ip == nil {
		return false
	}
	return !ip.IsLoopback() && !ip.IsPrivate() && !ip.IsUnspecified()
}

func labelsToJSON(labels map[string]string) string {
	if len(labels) == 0 {
		return "{}"
	}

	data, err := json.Marshal(labels)
	if err != nil {
		return "{}"
	}
	return string(data)
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
			if err := database.GetDB().Model(&model.TaskHostResult{}).Where("id = ?", hr.ID).Updates(updates).Error; err != nil {
				logger.Warn("update host result output failed", zap.Int64("host_result_id", hr.ID), zap.Error(err))
			}

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
			if err := database.GetDB().Model(&model.TaskHostResult{}).Where("id = ?", hr.ID).Updates(finalUpdates).Error; err != nil {
				logger.Warn("update host result final status failed", zap.Int64("host_result_id", hr.ID), zap.Error(err))
			}

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
// Uses optimistic locking (WHERE status IN ('pending','running')) to prevent concurrent double-updates.
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

	now := model.LocalTime(time.Now())
	finalStatus := "partial_fail"
	if failCount == 0 {
		finalStatus = "success"
	} else if successCount == 0 {
		finalStatus = "failed"
	}

	// Atomic update: only update if execution is still in a non-terminal state.
	// This prevents race conditions when multiple host results finish concurrently.
	result := database.GetDB().Model(&model.TaskExecution{}).
		Where("id = ? AND status IN ('pending','running')", executionID).
		Updates(map[string]interface{}{
			"status":        finalStatus,
			"finished_at":   &now,
			"success_count": successCount,
			"fail_count":    failCount,
		})
	if result.Error != nil {
		logger.Warn("更新执行状态失败", zap.Int64("execution_id", executionID), zap.Error(result.Error))
		return
	}
	if result.RowsAffected == 0 {
		// Already updated by another goroutine, skip
		return
	}

	logger.Info("Execution completed",
		zap.Int64("execution_id", executionID),
		zap.String("status", finalStatus),
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
		assetRepo: repository.NewAssetRepository(),
	})

	go func() {
		defer safego.Recover("grpc-server")
		logger.Info("gRPC server listening", zap.Int("port", port))
		if err := srv.Serve(lis); err != nil {
			logger.Error("gRPC server error", zap.Error(err))
		}
	}()

	return srv, nil
}
