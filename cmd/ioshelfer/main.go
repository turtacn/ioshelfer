package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	api "github.com/turtacn/ioshelfer/api/v1"
	"github.com/turtacn/ioshelfer/internal/common/logger"
	"github.com/turtacn/ioshelfer/internal/infra/ebpf"
)

// version 信息，编译时可通过 ldflags 设置
var (
	version   = "dev"
	commit    = "none"
	buildTime = "unknown"
)

func main() {
	// 定义命令行参数
	configPath := flag.String("config", "config.yaml", "Path to configuration file")
	showVersion := flag.Bool("version", false, "Show version information")
	flag.Parse()

	// 显示版本信息
	if *showVersion {
		fmt.Printf("ioshelfer version: %s\ncommit: %s\nbuild time: %s\n", version, commit, buildTime)
		os.Exit(0)
	}

	// 初始化日志
	log := logger.NewLogger()
	log.Info("Starting ioshelfer service...")

	// 初始化配置
	cfg, err := initConfig(*configPath, log)
	if err != nil {
		log.Fatalf("Failed to initialize config: %v", err)
	}

	// 初始化 eBPF 监控
	ebpfMonitor, err := ebpf.NewMonitor(cfg.EBPF, log)
	if err != nil {
		log.Fatalf("Failed to initialize eBPF monitor: %v", err)
	}
	defer ebpfMonitor.Close()

	// 初始化核心引擎
	engine, err := core.NewEngine(cfg.Core, ebpfMonitor, log)
	if err != nil {
		log.Fatalf("Failed to initialize core engine: %v", err)
	}

	// 初始化 API 服务
	apiServer, err := api.NewServer(cfg.API, engine, log)
	if err != nil {
		log.Fatalf("Failed to initialize API server: %v", err)
	}

	// 启动核心引擎
	if err := engine.Start(); err != nil {
		log.Fatalf("Failed to start core engine: %v", err)
	}

	// 启动 API 服务
	go func() {
		if err := apiServer.Start(); err != nil {
			log.Fatalf("API server failed: %v", err)
		}
	}()

	// 优雅退出处理
	gracefulShutdown(log, engine, apiServer)
}

// initConfig 初始化配置，支持配置文件和环境变量
func initConfig(configPath string, log *logrus.Logger) (*config.Config, error) {
	v := viper.New()

	// 设置配置文件路径
	v.SetConfigFile(configPath)
	v.SetConfigType("yaml")

	// 读取配置文件
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// 支持环境变量
	v.SetEnvPrefix("IOSHELFER")
	v.AutomaticEnv()

	// 默认配置
	v.SetDefault("api.port", 8080)
	v.SetDefault("ebpf.enabled", true)
	v.SetDefault("core.workers", 4)

	// 绑定配置到结构体
	var cfg config.Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	log.Infof("Configuration loaded: %+v", cfg)
	return &cfg, nil
}

// gracefulShutdown 处理优雅退出
func gracefulShutdown(log *logrus.Logger, engine *core.Engine, server *api.Server) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	log.Info("Received shutdown signal, stopping services...")

	// 停止核心引擎
	if err := engine.Stop(); err != nil {
		log.Errorf("Failed to stop core engine: %v", err)
	}

	// 停止 API 服务
	if err := server.Stop(); err != nil {
		log.Errorf("Failed to stop API server: %v", err)
	}

	log.Info("Services stopped successfully")
	os.Exit(0)
}
