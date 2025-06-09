package tests

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	api "github.com/turtacn/ioshelfer/api/v1"
	"github.com/turtacn/ioshelfer/internal/core/detection"
	"github.com/turtacn/ioshelfer/internal/core/prediction"
	"github.com/turtacn/ioshelfer/internal/infra/ebpf"
	"github.com/turtacn/ioshelfer/internal/common/logger"
	"github.com/turtacn/ioshelfer/pkg/disk"
	"github.com/turtacn/ioshelfer/pkg/network"
	"github.com/turtacn/ioshelfer/pkg/raid"
)

// TestMain 设置测试环境
func TestMain(m *testing.M) {
	// 初始化测试日志
	log := logger.NewLogger()
	log.SetLevel(logrus.DebugLevel)

	// 设置测试配置
	if err := setupTestConfig(); err != nil {
		log.Fatalf("Failed to setup test config: %v", err)
	}

	// 运行测试
	code := m.Run()

	// 清理测试环境
	cleanupTestEnvironment()

	os.Exit(code)
}

// setupTestConfig 创建测试配置文件
func setupTestConfig() error {
	v := viper.New()
	v.SetConfigType("yaml")
	v.SetConfigName("test-config")
	v.AddConfigPath(".")

	cfg := &config.Config{
		Core: config.CoreConfig{
			Detection: &config.DetectionConfig{
				Enabled:  true,
				Interval: 60,
				Devices:  []string{"/dev/testdisk"},
			},
			Prediction: &config.PredictionConfig{
				Model:      "test-model",
				Threshold:  0.7,
				TimeWindow: 30,
			},
		},
		EBPF: config.EBPFConfig{
			Enabled: true,
			Events:  []string{"disk_io", "network_io"},
		},
		API: config.APIConfig{
			Port: 18080,
		},
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal test config: %w", err)
	}

	return os.WriteFile("test-config.yaml", data, 0644)
}

// cleanupTestEnvironment 清理测试环境
func cleanupTestEnvironment() {
	os.Remove("test-config.yaml")
}

// TestIntegrationFullWorkflow 测试端到端工作流程
func TestIntegrationFullWorkflow(t *testing.T) {
	ctx := context.Background()
	log := logger.NewLogger()

	// 初始化配置
	cfg, err := loadTestConfig()
	require.NoError(t, err, "Failed to load test config")

	// 初始化 eBPF 监控
	ebpfMonitor, err := ebpf.NewMonitor(cfg.EBPF, log)
	require.NoError(t, err, "Failed to initialize eBPF monitor")
	defer ebpfMonitor.Close()

	// 初始化核心引擎
	engine, err := core.NewEngine(cfg.Core, ebpfMonitor, log)
	require.NoError(t, err, "Failed to initialize core engine")
	defer engine.Stop()

	// 初始化 API 服务
	apiServer, err := api.NewServer(cfg.API, engine, log)
	require.NoError(t, err, "Failed to initialize API server")
	go func() {
		if err := apiServer.Start(); err != nil {
			log.Errorf("API server failed: %v", err)
		}
	}()
	defer apiServer.Stop()

	// 等待 API 服务启动
	time.Sleep(1 * time.Second)

	t.Run("TestRAIDHealthCheck", func(t *testing.T) {
		testRAIDHealthCheck(t, ctx, engine)
	})

	t.Run("TestDiskHealthCheck", func(t *testing.T) {
		testDiskHealthCheck(t, ctx, engine)
	})

	t.Run("TestNetworkIOCheck", func(t *testing.T) {
		testNetworkIOCheck(t, ctx, engine)
	})

	t.Run("TestPredictionWorkflow", func(t *testing highQueueDepthScenario(t, ctx, engine)
})

t.Run("TestRepairWorkflow", func(t *testing.T) {
	testRepairWorkflow(t, ctx, engine)
})
}

// loadTestConfig 加载测试配置
func loadTestConfig() (*config.Config, error) {
	v := viper.New()
	v.SetConfigFile("test-config.yaml")
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read test config: %w", err)
	}

	var cfg config.Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal test config: %w", err)
	}
	return &cfg, nil
}

// testRAIDHealthCheck 测试 RAID 健康检查
func testRAIDHealthCheck(t *testing.T, ctx context.Context, engine *core.Engine) {
	// 模拟 RAID 控制器
	raidController := raid.NewMockController()
	raidController.SimulateArray("raid0", []string{"/dev/testdisk1", "/dev/testdisk2"})
	raidController.InjectError("raid0", raid.ErrorTypeDegraded)

	// 初始化检测器
	detector, err := detection.NewDetector(engine.Config.Detection, logrus.New())
	require.NoError(t, err, "Failed to initialize detector")

	// 执行 RAID 健康检查
	result, err := detector.CheckRAID(ctx, "raid0")
	require.NoError(t, err, "RAID health check failed")
	assert.Equal(t, "degraded", result.Status, "Expected degraded RAID status")
	assert.Contains(t, result.Issues, "Array degraded due to disk failure", "Expected degraded issue")
}

// testDiskHealthCheck 测试磁盘健康检查
func testDiskHealthCheck(t *testing.T, ctx context.Context, engine *core.Engine) {
	// 模拟磁盘
	diskSimulator := disk.NewMockDisk("/dev/testdisk")
	diskSimulator.InjectSMARTError("Reallocated_Sector_Ct", 100)

	// 初始化检测器
	detector, err := detection.NewDetector(engine.Config.Detection, logrus.New())
	require.NoError(t, err, "Failed to initialize detector")

	// 执行磁盘健康检查
	result, err := detector.CheckDevice(ctx, "/dev/testdisk", true)
	require.NoError(t, err, "Disk health check failed")
	assert.True(t, result.Score < 0.8, "Expected health score < 0.8 due to SMART error")
	assert.Contains(t, result.Issues, "High reallocated sector count", "Expected SMART error issue")
}

// testNetworkIOCheck 测试网络 I/O 检查
func testNetworkIOCheck(t *testing.T, ctx context.Context, engine *core.Engine) {
	// 模拟网络接口
	networkSimulator := network.NewMockInterface("eth0")
	networkSimulator.InjectHighQueueDepth(1000)

	// 初始化检测器
	detector, err := detection.NewDetector(engine.Config.Detection, logrus.New())
	require.NoError(t, err, "Failed to initialize detector")

	// 执行网络 I/O 检查
	result, err := detector.CheckNetworkIO(ctx, "eth0")
	require.NoError(t, err, "Network I/O check failed")
	assert.True(t, result.QueueDepth > 500, "Expected high queue depth")
	assert.Contains(t, result.Issues, "High queue depth detected", "Expected queue depth issue")
}

// testPredictionWorkflow 测试预测工作流程
func testPredictionWorkflow(t *testing.T, ctx context.Context, engine *core.Engine) {
	// 模拟磁盘亚健康场景
	diskSimulator := disk.NewMockDisk("/dev/testdisk")
	diskSimulator.InjectSMARTError("Reallocated_Sector_Ct", 150)
	diskSimulator.InjectSMARTError("Current_Pending_Sector", 50)

	// 初始化预测器
	predictor, err := prediction.NewPredictor(engine.Config.Prediction, logrus.New())
	require.NoError(t, err, "Failed to initialize predictor")

	// 执行预测
	result, err := predictor.Predict("/dev/testdisk", "test-model", 30)
	require.NoError(t, err, "Prediction failed")
	assert.True(t, result.Probability > 0.5, "Expected high failure probability")
	assert.Equal(t, "high", result.RiskLevel, "Expected high risk level")
	assert.Contains(t, result.ContributingFactors, "High reallocated sector count", "Expected reallocated sector factor")
}

// testRepairWorkflow 测试修复工作流程
func testRepairWorkflow(t *testing.T, ctx context.Context, engine *core.Engine) {
	// 模拟 RAID 阵列和磁盘错误
	raidController := raid.NewMockController()
	raidController.SimulateArray("raid0", []string{"/dev/testdisk1", "/dev/testdisk2"})
	raidController.InjectError("raid0", raid.ErrorTypeDegraded)

	// 初始化检测器
	detector, err := detection.NewDetector(engine.Config.Detection, logrus.New())
	require.NoError(t, err, "Failed to initialize detector")

	// 检测问题
	result, err := detector.CheckRAID(ctx, "raid0")
	require.NoError(t, err, "RAID check failed")
	assert.Equal(t, "degraded", result.Status, "Expected degraded status")

	// 执行修复
	err = raidController.RepairArray("raid0")
	require.NoError(t, err, "RAID repair failed")

	// 验证修复结果
	result, err = detector.CheckRAID(ctx, "raid0")
	require.NoError(t, err, "Post-repair RAID check failed")
	assert.Equal(t, "healthy", result.Status, "Expected healthy status after repair")
}

// TestPerformance 测试系统性能
func TestPerformance(t *testing.T) {
	ctx := context.Background()
	log := logger.NewLogger()

	// 初始化配置和组件
	cfg, err := loadTestConfig()
	require.NoError(t, err, "Failed to load test config")

	ebpfMonitor, err := ebpf.NewMonitor(cfg.EBPF, log)
	require.NoError(t, err, "Failed to initialize eBPF monitor")
	defer ebpfMonitor.Close()

	engine, err := core.NewEngine(cfg.Core, ebpfMonitor, log)
	require.NoError(t, err, "Failed to initialize core engine")
	defer engine.Stop()

	// 模拟高负载场景
	diskSimulator := disk.NewMockDisk("/dev/testdisk")
	diskSimulator.InjectHighIO(1000) // 模拟高 I/O 负载

	// 测试检测性能
	start := time.Now()
	detector, err := detection.NewDetector(cfg.Core.Detection, log)
	require.NoError(t, err, "Failed to initialize detector")

	_, err = detector.CheckDevice(ctx, "/dev/testdisk", true)
	require.NoError(t, err, "Disk check failed under load")
	duration := time.Since(start)
	assert.True(t, duration < 5*time.Second, "Disk check took too long: %v", duration)

	// 测试预测性能
	predictor, err := prediction.NewPredictor(cfg.Core.Prediction, log)
	require.NoError(t, err, "Failed to initialize predictor")

	start = time.Now()
	_, err = predictor.Predict("/dev/testdisk", "test-model", 30)
	require.NoError(t, err, "Prediction failed under load")
	duration = time.Since(start)
	assert.True(t, duration < 2*time.Second, "Prediction took too long: %v", duration)
}
