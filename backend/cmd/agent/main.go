package main

import (
	"context"
	"flag"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/bigops/platform/internal/agent"
	"github.com/spf13/viper"
)

func main() {
	configPath := flag.String("config", "config/agent.yaml", "config file path")
	flag.Parse()

	viper.SetConfigFile(*configPath)
	viper.SetConfigType("yaml")
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Failed to read config: %v", err)
	}

	serverAddr := viper.GetString("server.address")
	hostname := viper.GetString("agent.hostname")
	if hostname == "" {
		h, _ := os.Hostname()
		hostname = h
	}

	ip := getLocalIP()

	log.Printf("BigOps Agent starting, server=%s hostname=%s ip=%s", serverAddr, hostname, ip)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client := agent.NewAgentClient(serverAddr, hostname, ip)
	if err := client.Connect(); err != nil {
		log.Fatalf("Failed to connect to server: %v", err)
	}
	defer client.Close()

	go client.Run(ctx)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Agent shutting down...")
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
