package main

import (
	"context"
	"flag"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bigops/platform/internal/agent"
	"github.com/bigops/platform/internal/pkg/config"
	"github.com/bigops/platform/internal/pkg/logger"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

func main() {
	configPath := flag.String("config", "config/agent.yaml", "config file path")
	flag.Parse()

	viper.SetConfigFile(*configPath)
	viper.SetConfigType("yaml")
	if err := viper.ReadInConfig(); err != nil {
		logger.Fatal("Failed to read config", zap.Error(err))
	}

	// 使用统一配置加载（取代直接viper调用）
	if err := config.Load(*configPath); err != nil {
		logger.Fatal("Failed to load config via config package", zap.Error(err))
	}
	cfg := config.Get()
	agentCfg := config.GetAgentConfig()

	serverAddr := cfg.Server.Address
	hostname := agentCfg.Hostname
	configuredAgentID := agentCfg.ID
	configuredStateFile := agentCfg.StateFile
	publicIP := agentCfg.PublicIP
	publicIPProvider := agentCfg.PublicIPProvider
	publicIPCacheFile := agentCfg.PublicIPCacheFile
	publicIPTimeoutSeconds := agentCfg.PublicIPTimeoutSeconds
	publicIPRefreshHours := agentCfg.PublicIPRefreshHours

	// Resource limits (inspired by didi/falcon-log-agent)
	maxCPURate := agentCfg.MaxCPURate
	maxMemMB := agentCfg.MaxMemMB
	if maxCPURate > 0 {
		cores := agent.ApplyCPULimit(maxCPURate)
		logger.Info("CPU limit applied",
			zap.String("agent_id", configuredAgentID),
			zap.Int("gomaxprocs", cores),
			zap.Float64("rate", maxCPURate),
		)
	}
	if maxMemMB > 0 {
		agent.StartMemoryPatrol(configuredAgentID, maxMemMB, 10*time.Second)
		logger.Info("Memory patrol started",
			zap.String("agent_id", configuredAgentID),
			zap.Int("limit_mb", maxMemMB),
		)
	}
	if hostname == "" {
		h, _ := os.Hostname()
		hostname = h
	}

	privateIP := getLocalIP()
	stateFile := resolveAgentStateFile(*configPath, configuredStateFile)
	publicIPCachePath := resolvePublicIPCacheFile(*configPath, publicIPCacheFile)
	agentID, err := resolveAgentID(configuredAgentID, stateFile)
	if err != nil {
		logger.Fatal("Failed to resolve agent id", zap.Error(err))
	}
	publicIPCfg := publicIPConfig{
		ConfiguredPublicIP: publicIP,
		CacheFile:          publicIPCachePath,
		ProviderURL:        publicIPProvider,
		Timeout:            time.Duration(publicIPTimeoutSeconds) * time.Second,
		RefreshInterval:    time.Duration(publicIPRefreshHours) * time.Hour,
		CheckInterval:      time.Hour,
	}
	resolvedPublicIP, err := resolvePublicIP(publicIPCfg)
	if err != nil {
		logger.Warn("Resolve public ip failed",
			zap.String("agent_id", agentID),
			zap.Error(err),
		)
	}
	publicIP = resolvedPublicIP

	logger.Info("BigOps Agent starting",
		zap.String("agent_id", agentID),
		zap.String("server", serverAddr),
		zap.String("hostname", hostname),
		zap.String("private_ip", privateIP),
		zap.String("public_ip", publicIP),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client := agent.NewAgentClient(serverAddr, agentID, hostname, privateIP, publicIP)
	if err := client.Connect(); err != nil {
		logger.Fatal("Failed to connect to server",
			zap.String("agent_id", agentID),
			zap.String("server", serverAddr),
			zap.Error(err),
		)
	}
	defer client.Close()

	go client.Run(ctx)
	go watchPublicIP(ctx, publicIPCfg, client)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("Agent shutting down",
		zap.String("agent_id", agentID),
	)
	cancel()
}

func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "127.0.0.1"
	}
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && ipnet.IP.To4() != nil {
			return ipnet.IP.String()
		}
	}
	return "127.0.0.1"
}
