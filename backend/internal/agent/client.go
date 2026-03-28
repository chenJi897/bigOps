package agent

import (
	"context"
	"fmt"
	"log"
	"runtime"
	"sync"
	"time"

	pb "github.com/bigops/platform/proto/gen/agent"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	agentVersion      = "1.0.0"
	heartbeatInterval = 10 * time.Second
	reconnectDelay    = 5 * time.Second
)

type AgentClient struct {
	serverAddr string
	hostname   string
	privateIP  string
	publicIP   string
	agentID    string
	metrics    *MetricsCollector

	conn   *grpc.ClientConn
	client pb.AgentServiceClient
	mu     sync.Mutex
}

func NewAgentClient(serverAddr, agentID, hostname, privateIP, publicIP string) *AgentClient {
	return &AgentClient{
		serverAddr: serverAddr,
		hostname:   hostname,
		privateIP:  privateIP,
		publicIP:   publicIP,
		agentID:    agentID,
		metrics:    NewMetricsCollector(),
	}
}

func (c *AgentClient) Connect() error {
	conn, err := grpc.NewClient(c.serverAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return fmt.Errorf("grpc dial failed: %w", err)
	}
	c.conn = conn
	c.client = pb.NewAgentServiceClient(conn)
	log.Printf("Connected to server %s", c.serverAddr)
	return nil
}

func (c *AgentClient) Close() {
	if c.conn != nil {
		c.conn.Close()
	}
}

func (c *AgentClient) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		err := c.heartbeatLoop(ctx)
		if err != nil {
			log.Printf("Heartbeat stream ended: %v, reconnecting in %s...", err, reconnectDelay)
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(reconnectDelay):
		}
	}
}

func (c *AgentClient) heartbeatLoop(ctx context.Context) error {
	stream, err := c.client.Heartbeat(ctx)
	if err != nil {
		return fmt.Errorf("open heartbeat stream: %w", err)
	}

	// Goroutine to receive server responses (including task assignments)
	executor := NewExecutor()
	go func() {
		for {
			resp, err := stream.Recv()
			if err != nil {
				log.Printf("Heartbeat recv error: %v", err)
				return
			}
			if resp.Task != nil {
				log.Printf("Received task: execution_id=%d host_result_id=%d",
					resp.Task.ExecutionId, resp.Task.HostResultId)
				go c.executeTask(ctx, executor, resp.Task)
			}
		}
	}()

	// Send heartbeats periodically
	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()

	// Send initial heartbeat immediately
	if err := c.sendHeartbeat(stream); err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := c.sendHeartbeat(stream); err != nil {
				return err
			}
		}
	}
}

func (c *AgentClient) sendHeartbeat(stream pb.AgentService_HeartbeatClient) error {
	metrics := c.metrics.Collect()
	publicIP := c.getPublicIP()

	req := &pb.HeartbeatRequest{
		AgentId:            c.agentID,
		Hostname:           c.hostname,
		Ip:                 c.privateIP,
		PrivateIp:          c.privateIP,
		PublicIp:           publicIP,
		Version:            agentVersion,
		Os:                 runtime.GOOS + "/" + runtime.GOARCH,
		CpuCount:           metrics.CPUCount,
		CpuUsagePercent:    metrics.CPUUsagePct,
		MemoryTotal:        metrics.MemoryTotal,
		MemoryUsed:         metrics.MemoryUsed,
		MemoryUsagePercent: metrics.MemoryUsagePct,
		DiskTotal:          metrics.DiskTotal,
		DiskUsed:           metrics.DiskUsed,
		DiskUsagePercent:   metrics.DiskUsagePct,
		Timestamp:          time.Now().Unix(),
	}
	return stream.Send(req)
}

func (c *AgentClient) SetPublicIP(ip string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.publicIP = ip
}

func (c *AgentClient) getPublicIP() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.publicIP
}

func (c *AgentClient) executeTask(ctx context.Context, executor *Executor, task *pb.ExecuteRequest) {
	log.Printf("Executing task: host_result_id=%d script_type=%s",
		task.HostResultId, task.ScriptType)

	// Open ReportOutput stream to send results back
	reportStream, err := c.client.ReportOutput(ctx)
	if err != nil {
		log.Printf("Failed to open report stream: %v", err)
		return
	}

	// Execute and stream output
	executor.Execute(ctx, task, reportStream)

	_, err = reportStream.CloseAndRecv()
	if err != nil {
		log.Printf("Failed to close report stream: %v", err)
	}
}
