package grpc

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"

	pb "github.com/bigops/platform/proto/gen/agent"

	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/pkg/logger"
	"github.com/bigops/platform/internal/repository"
)

// AgentStream represents a connected agent's bidirectional heartbeat stream.
type AgentStream struct {
	AgentID  string
	Hostname string
	IP       string
	Stream   grpc.BidiStreamingServer[pb.HeartbeatRequest, pb.HeartbeatResponse]
	Cancel   context.CancelFunc
}

// LogLine is a single line of execution output for WebSocket streaming.
type LogLine struct {
	HostResultID int64  `json:"host_result_id"`
	HostIP       string `json:"host_ip"`
	Line         string `json:"line"`
	IsStderr     bool   `json:"is_stderr"`
	Phase        string `json:"phase"`
	ExitCode     int32  `json:"exit_code"`
	Timestamp    int64  `json:"timestamp"`
}

// AgentManager tracks connected agents and manages WebSocket log subscriptions.
type AgentManager struct {
	mu     sync.RWMutex
	agents map[string]*AgentStream // key: agentID

	repo *repository.AgentRepository

	// WebSocket log subscribers: executionID -> list of channels
	logSubsMu sync.RWMutex
	logSubs   map[int64][]chan *LogLine
}

var manager *AgentManager
var managerOnce sync.Once

// InitAgentManager initializes the singleton with the given repository.
func InitAgentManager(repo *repository.AgentRepository) {
	managerOnce.Do(func() {
		manager = &AgentManager{
			agents:  make(map[string]*AgentStream),
			repo:    repo,
			logSubs: make(map[int64][]chan *LogLine),
		}
	})
}

// GetAgentManager returns the global AgentManager singleton.
func GetAgentManager() *AgentManager {
	if manager == nil {
		// Fallback init (shouldn't happen after proper boot)
		InitAgentManager(repository.NewAgentRepository())
	}
	return manager
}

// RegisterAgent records a connected agent in memory and updates DB status to online.
func (m *AgentManager) RegisterAgent(agentID string, stream grpc.BidiStreamingServer[pb.HeartbeatRequest, pb.HeartbeatResponse], cancel context.CancelFunc, hostname, ip string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// If agent was already connected, cancel old stream
	if old, ok := m.agents[agentID]; ok {
		old.Cancel()
	}

	m.agents[agentID] = &AgentStream{
		AgentID:  agentID,
		Hostname: hostname,
		IP:       ip,
		Stream:   stream,
		Cancel:   cancel,
	}

	now := model.LocalTime(time.Now())
	_ = m.repo.Upsert(&model.AgentInfo{
		AgentID:       agentID,
		Hostname:      hostname,
		IP:            ip,
		Status:        "online",
		LastHeartbeat: &now,
	})

	logger.Info("Agent registered", zap.String("agent_id", agentID), zap.String("ip", ip))
}

// UnregisterAgent removes agent from memory and marks it offline in DB.
func (m *AgentManager) UnregisterAgent(agentID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.agents, agentID)
	_ = m.repo.UpdateStatus(agentID, "offline")

	logger.Info("Agent unregistered", zap.String("agent_id", agentID))
}

// GetAgentByIP returns the connected agent stream for a given IP.
func (m *AgentManager) GetAgentByIP(ip string) *AgentStream {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, a := range m.agents {
		if a.IP == ip {
			return a
		}
	}
	return nil
}

// GetAgent returns the connected agent stream for a given agentID.
func (m *AgentManager) GetAgent(agentID string) *AgentStream {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.agents[agentID]
}

// SendTaskToAgent sends a task to a specific agent via its heartbeat stream.
func (m *AgentManager) SendTaskToAgent(agentID string, task *pb.ExecuteRequest) error {
	m.mu.RLock()
	agent, ok := m.agents[agentID]
	m.mu.RUnlock()

	if !ok {
		return fmt.Errorf("agent %s not connected", agentID)
	}

	resp := &pb.HeartbeatResponse{
		Accepted:   true,
		ServerTime: time.Now().Unix(),
		Task:       task,
	}
	if err := agent.Stream.Send(resp); err != nil {
		return fmt.Errorf("failed to send task to agent %s: %w", agentID, err)
	}
	return nil
}

// OnlineCount returns the number of currently connected agents.
func (m *AgentManager) OnlineCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.agents)
}

// --- WebSocket Log Pub/Sub ---

// SubscribeLogs creates a channel that will receive log lines for the given execution.
func (m *AgentManager) SubscribeLogs(executionID int64) chan *LogLine {
	m.logSubsMu.Lock()
	defer m.logSubsMu.Unlock()

	ch := make(chan *LogLine, 256)
	m.logSubs[executionID] = append(m.logSubs[executionID], ch)
	return ch
}

// UnsubscribeLogs removes a subscriber channel for the given execution.
func (m *AgentManager) UnsubscribeLogs(executionID int64, ch chan *LogLine) {
	m.logSubsMu.Lock()
	defer m.logSubsMu.Unlock()

	subs := m.logSubs[executionID]
	for i, sub := range subs {
		if sub == ch {
			m.logSubs[executionID] = append(subs[:i], subs[i+1:]...)
			break
		}
	}
	if len(m.logSubs[executionID]) == 0 {
		delete(m.logSubs, executionID)
	}
	close(ch)
}

// PublishLog sends a log line to all subscribers of the given execution.
func (m *AgentManager) PublishLog(executionID int64, line *LogLine) {
	m.logSubsMu.RLock()
	subs := m.logSubs[executionID]
	m.logSubsMu.RUnlock()

	for _, ch := range subs {
		select {
		case ch <- line:
		default:
			// Channel full, skip to prevent blocking
		}
	}
}
