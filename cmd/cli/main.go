package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/turtacn/ioshelfer/internal"
	"github.com/turtacn/ioshelfer/internal/core/detection"
	"github.com/turtacn/ioshelfer/internal/core/prediction"
	"github.com/turtacn/ioshelfer/internal/common/logger"
	"github.com/turtacn/ioshelfer/internal/infra/storage"
)

// 版本信息
var (
	version   = "dev"
	commit    = "none"
	buildTime = "unknown"
)

// 全局配置和组件
var (
	cfgFile    string
	cfg        *config.Config
	detector   *detection.Detector
	predictor  *prediction.Predictor
	log        = logger.NewLogger()
)

// rootCmd 根命令
var rootCmd = &cobra.Command{
	Use:   "ioshelfer-cli",
	Short: "IOShelfer command line interface",
	Long: `IOShelfer CLI provides command line tools for disk failure prediction
and system health monitoring. You can perform manual checks, predictions,
and view system status.`,
	Version: fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, buildTime),
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// 全局标志
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.ioshelfer.yaml)")
	rootCmd.PersistentFlags().Bool("verbose", false, "verbose output")
	rootCmd.PersistentFlags().String("format", "table", "output format (table, json, yaml)")

	// 绑定标志到配置
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
	viper.BindPFlag("format", rootCmd.PersistentFlags().Lookup("format"))

	// 添加子命令
	rootCmd.AddCommand(healthcheckCmd)
	rootCmd.AddCommand(predictCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(configCmd)
}

// initConfig 初始化配置
func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			log.Fatal(err)
		}

		viper.AddConfigPath(home)
		viper.AddConfigPath(".")
		viper.SetConfigType("yaml")
		viper.SetConfigName(".ioshelfer")
	}

	viper.AutomaticEnv()
	viper.SetEnvPrefix("IOSHELFER")

	if err := viper.ReadInConfig(); err == nil {
		log.Debugf("Using config file: %s", viper.ConfigFileUsed())
	}

	// 解析配置
	if err := viper.Unmarshal(&cfg); err != nil {
		log.Fatalf("Failed to unmarshal config: %v", err)
	}

	// 初始化组件
	initComponents()
}

// initComponents 初始化检测和预测组件
func initComponents() {
	var err error

	// 初始化检测器
	detector, err = detection.NewDetector(cfg.Core.Detection, log)
	if err != nil {
		log.Warnf("Failed to initialize detector: %v", err)
	}

	// 初始化预测器
	predictor, err = prediction.NewPredictor(cfg.Core.Prediction, log)
	if err != nil {
		log.Warnf("Failed to initialize predictor: %v", err)
	}
}

// healthcheckCmd 健康检查命令
var healthcheckCmd = &cobra.Command{
	Use:   "healthcheck [device]",
	Short: "Perform health check on disk devices",
	Long: `Perform a comprehensive health check on specified disk device or all devices.
This command will check SMART data, run basic diagnostics, and report current status.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runHealthcheck,
}

func init() {
	healthcheckCmd.Flags().Bool("all", false, "check all devices")
	healthcheckCmd.Flags().Bool("smart", true, "include SMART data analysis")
	healthcheckCmd.Flags().Bool("detailed", false, "show detailed results")
	healthcheckCmd.Flags().Duration("timeout", 30*time.Second, "check timeout")
}

func runHealthcheck(cmd *cobra.Command, args []string) error {
	if detector == nil {
		return fmt.Errorf("detector not initialized")
	}

	ctx, cancel := context.WithTimeout(context.Background(), getTimeout(cmd))
	defer cancel()

	all, _ := cmd.Flags().GetBool("all")
	smart, _ := cmd.Flags().GetBool("smart")
	detailed, _ := cmd.Flags().GetBool("detailed")

	var devices []string
	if all {
		// 获取所有设备
		devices = getAllDevices()
	} else if len(args) > 0 {
		devices = []string{args[0]}
	} else {
		devices = getDefaultDevices()
	}

	results := make([]*HealthCheckResult, 0)

	for _, device := range devices {
		result, err := performHealthCheck(ctx, device, smart, detailed)
		if err != nil {
			log.Errorf("Health check failed for %s: %v", device, err)
			continue
		}
		results = append(results, result)
	}

	return outputResults(cmd, "healthcheck", results)
}

// predictCmd 预测命令
var predictCmd = &cobra.Command{
	Use:   "predict [device]",
	Short: "Predict disk failure probability",
	Long: `Analyze disk metrics and predict failure probability using machine learning models.
This command provides failure risk assessment and recommended actions.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runPredict,
}

func init() {
	predictCmd.Flags().Bool("all", false, "predict for all devices")
	predictCmd.Flags().String("model", "default", "prediction model to use")
	predictCmd.Flags().Int("days", 30, "prediction time horizon in days")
	predictCmd.Flags().Bool("explain", false, "include prediction explanation")
}

func runPredict(cmd *cobra.Command, args []string) error {
	if predictor == nil {
		return fmt.Errorf("predictor not initialized")
	}

	all, _ := cmd.Flags().GetBool("all")
	model, _ := cmd.Flags().GetString("model")
	days, _ := cmd.Flags().GetInt("days")
	explain, _ := cmd.Flags().GetBool("explain")

	var devices []string
	if all {
		devices = getAllDevices()
	} else if len(args) > 0 {
		devices = []string{args[0]}
	} else {
		devices = getDefaultDevices()
	}

	results := make([]*PredictionResult, 0)

	for _, device := range devices {
		result, err := performPrediction(device, model, days, explain)
		if err != nil {
			log.Errorf("Prediction failed for %s: %v", device, err)
			continue
		}
		results = append(results, result)
	}

	return outputResults(cmd, "prediction", results)
}

// statusCmd 状态命令
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show system and service status",
	Long: `Display current system status, service health, and recent alerts.
This provides an overview of the IOShelfer monitoring system.`,
	RunE: runStatus,
}

func runStatus(cmd *cobra.Command, args []string) error {
	status := &SystemStatus{
		Timestamp: time.Now(),
		Service: ServiceStatus{
			Running: true,
			Version: version,
			Uptime:  "unknown",
		},
		Devices:    getDeviceStatuses(),
		Alerts:     getRecentAlerts(),
		Statistics: getSystemStatistics(),
	}

	return outputResults(cmd, "status", status)
}

// listCmd 列表命令
var listCmd = &cobra.Command{
	Use:   "list [resource]",
	Short: "List devices, alerts, or predictions",
	Long: `List various resources like devices, alerts, predictions, or models.
Available resources: devices, alerts, predictions, models`,
	Args: cobra.ExactArgs(1),
	RunE: runList,
}

func runList(cmd *cobra.Command, args []string) error {
	resource := args[0]

	switch resource {
	case "devices":
		return listDevices(cmd)
	case "alerts":
		return listAlerts(cmd)
	case "predictions":
		return listPredictions(cmd)
	case "models":
		return listModels(cmd)
	default:
		return fmt.Errorf("unknown resource: %s", resource)
	}
}

// configCmd 配置命令
var configCmd = &cobra.Command{
	Use:   "config [action]",
	Short: "Manage configuration",
	Long: `Manage IOShelfer configuration. Actions: view, validate, generate-sample`,
	Args: cobra.ExactArgs(1),
	RunE: runConfig,
}

func runConfig(cmd *cobra.Command, args []string) error {
	action := args[0]

	switch action {
	case "view":
		return viewConfig(cmd)
	case "validate":
		return validateConfig(cmd)
	case "generate-sample":
		return generateSampleConfig(cmd)
	default:
		return fmt.Errorf("unknown action: %s", action)
	}
}

// 辅助函数和结构体定义

type HealthCheckResult struct {
	Device      string                 `json:"device"`
	Status      string                 `json:"status"`
	Score       float64                `json:"score"`
	Issues      []string               `json:"issues,omitempty"`
	SMARTData   map[string]interface{} `json:"smart_data,omitempty"`
	Temperature int                    `json:"temperature,omitempty"`
	CheckTime   time.Time              `json:"check_time"`
}

type PredictionResult struct {
	Device           string    `json:"device"`
	FailureProbability float64 `json:"failure_probability"`
	RiskLevel        string    `json:"risk_level"`
	TimeHorizon      int       `json:"time_horizon_days"`
	Factors          []string  `json:"contributing_factors,omitempty"`
	Recommendations  []string  `json:"recommendations,omitempty"`
	PredictionTime   time.Time `json:"prediction_time"`
}

type SystemStatus struct {
	Timestamp  time.Time           `json:"timestamp"`
	Service    ServiceStatus       `json:"service"`
	Devices    []DeviceStatus      `json:"devices"`
	Alerts     []models.Alert      `json:"recent_alerts"`
	Statistics SystemStatistics    `json:"statistics"`
}

type ServiceStatus struct {
	Running bool   `json:"running"`
	Version string `json:"version"`
	Uptime  string `json:"uptime"`
}

type DeviceStatus struct {
	Device      string  `json:"device"`
	Status      string  `json:"status"`
	Health      float64 `json:"health_score"`
	Temperature int     `json:"temperature"`
	LastCheck   time.Time `json:"last_check"`
}

type SystemStatistics struct {
	TotalDevices      int `json:"total_devices"`
	HealthyDevices    int `json:"healthy_devices"`
	WarningDevices    int `json:"warning_devices"`
	CriticalDevices   int `json:"critical_devices"`
	ActiveAlerts      int `json:"active_alerts"`
	PredictionsToday  int `json:"predictions_today"`
}

// 实现具体的功能函数
func performHealthCheck(ctx context.Context, device string, includeSMART, detailed bool) (*HealthCheckResult, error) {
	result := &HealthCheckResult{
		Device:    device,
		CheckTime: time.Now(),
	}

	// 执行健康检查
	health, err := detector.CheckDevice(ctx, device, includeSMART)
	if err != nil {
		return nil, fmt.Errorf("failed to check device %s: %w", device, err)
	}

	result.Status = health.Status
	result.Score = health.Score
	result.Temperature = health.Temperature
	if detailed {
		result.Issues = health.Issues
		if includeSMART {
			result.SMARTData = health.SMARTData
		}
	}

	return result, nil
}

func performPrediction(device, model string, days int, explain bool) (*PredictionResult, error) {
	result := &PredictionResult{
		Device:         device,
		TimeHorizon:    days,
		PredictionTime: time.Now(),
	}

	// 执行预测
	prediction, err := predictor.Predict(device, model, days)
	if err != nil {
		return nil, fmt.Errorf("failed to predict for device %s: %w", device, err)
	}

	result.FailureProbability = prediction.Probability
	result.RiskLevel = prediction.RiskLevel
	if explain {
		result.Factors = prediction.ContributingFactors
		result.Recommendations = prediction.Recommendations
	}

	return result, nil
}

func listDevices(cmd *cobra.Command) error {
	devices, err := detector.ListDevices()
	if err != nil {
		return fmt.Errorf("failed to list devices: %w", err)
	}

	return outputResults(cmd, "devices", devices)
}

func listAlerts(cmd *cobra.Command) error {
	alerts, err := models.GetRecentAlerts()
	if err != nil {
		return fmt.Errorf("failed to list alerts: %w", err)
	}

	return outputResults(cmd, "alerts", alerts)
}

func listPredictions(cmd *cobra.Command) error {
	predictions, err := models.GetRecentPredictions()
	if err != nil {
		return fmt.Errorf("failed to list predictions: %w", err)
	}

	return outputResults(cmd, "predictions", predictions)
}

func listModels(cmd *cobra.Command) error {
	models, err := predictor.ListModels()
	if err != nil {
		return fmt.Errorf("failed to list models: %w", err)
	}

	return outputResults(cmd, "models", models)
}

func viewConfig(cmd *cobra.Command) error {
	configData, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	fmt.Println(string(configData))
	return nil
}

func validateConfig(cmd *cobra.Command) error {
	if cfg == nil {
		return fmt.Errorf("configuration not loaded")
	}

	// 简单的配置验证逻辑
	if cfg.Core.Detection == nil || cfg.Core.Prediction == nil {
		return fmt.Errorf("incomplete configuration: detection or prediction settings missing")
	}

	fmt.Println("Configuration is valid")
	return nil
}

func generateSampleConfig(cmd *cobra.Command) error {
	sampleCfg := &config.Config{
		Core: config.CoreConfig{
			Detection: &config.DetectionConfig{
				Enabled: true,
				Interval: 3600, // 1 hour
			},
			Prediction: &config.PredictionConfig{
				Model:      "default",
				Threshold:  0.7,
				TimeWindow: 30, // days
			},
		},
	}

	sampleData, err := json.MarshalIndent(sampleCfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to generate sample config: %w", err)
	}

	samplePath := "sample-config.json"
	if err := os.WriteFile(samplePath, sampleData, 0644); err != nil {
		return fmt.Errorf("failed to write sample config: %w", err)
	}

	fmt.Printf("Sample configuration generated at %s\n", samplePath)
	return nil
}

func getAllDevices() []string {
	devices, err := detector.ListDevices()
	if err != nil {
		log.Errorf("Failed to get all devices: %v", err)
		return []string{}
	}
	return devices
}

func getDefaultDevices() []string {
	// 默认返回配置中的设备列表或单个默认设备
	if cfg.Core.Detection != nil && len(cfg.Core.Detection.Devices) > 0 {
		return cfg.Core.Detection.Devices
	}
	return []string{"/dev/sda"} // 默认设备
}

func getDeviceStatuses() []HttpCheckResult {
	// 模拟设备状态，实际实现应从 detector 获取
	return []DeviceStatus{
		{
			Device:      "/dev/sda",
			Ascertain:   true,
			Status:      "healthy",
			Health:      0.95,
			Temperature: 35,
			LastCheck:   time.Now(),
		},
	}
}

func getRecentAlerts() []models.Alert {
	// 模拟从存储获取最近的告警
	return []models.Alert{
		{
			ID:        1,
			Device:    "/dev/sda",
			Severity:  "warning",
			Message:   "High reallocated sector count",
			Timestamp: time.Now().Add(-time.Hour),
		},
	}
}

func getSystemStatistics() SystemStatistics {
	// 模拟系统统计数据
	return SystemStatistics{
		TotalDevices:     2,
		HealthyDevices:   1,
		WarningDevices:   1,
		CriticalDevices:  0,
		ActiveAlerts:     1,
		PredictionsToday: 5,
	}
}

func getTimeout(cmd *cobra.Command) time.Duration {
	timeout, _ := cmd.Flags().GetDuration("timeout")
	return timeout
}

func outputResults(cmd *cobra.Command, resultType string, data interface{}) error {
	format, _ := cmd.Flags().GetString("format")
	verbose, _ := cmd.Flags().GetBool("verbose")

	var output []byte
	var err error

	switch format {
	case "json":
		if verbose {
			output, err = json.MarshalIndent(data, "", "  ")
		} else {
			output, err = json.Marshal(data)
		}
	case "yaml":
		output, err = yaml.Marshal(data)
	case "table":
		return outputTable(resultType, data)
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}

	if err != nil {
		return fmt.Errorf("failed to marshal output: %w", err)
	}

	fmt.Println(string(output))
	return nil
}

func outputTable(resultType string, data interface{}) error {
	// 使用 tablewriter 或类似库格式化表格输出
	// 这里简化为简单打印，实际实现可添加 tablewriter 依赖
	switch resultType {
	case "healthcheck":
		results, ok := data.([]*HealthCheckResult)
		if !ok {
			return fmt.Errorf("invalid data type for healthcheck")
		}
		for _, r := range results {
			fmt.Printf("Device: %s, Status: %s, Score: %.2f, Time: %s\n", r.Device, r.Status, r.Score, r.CheckTime.Format(time.RFC3339))
		}
	case "prediction":
		results, ok := data.([]*PredictionResult)
		if !ok {
			return fmt.Errorf("invalid data type for prediction")
		}
		for _, r := range results {
			fmt.Printf("Device: %s, Probability: %.2f%%, Risk: %s, Time: %s\n", r.Device, r.FailureProbability*100, r.RiskLevel, r.PredictionTime.Format(time.RFC3339))
		}
	case "status":
		status, ok := data.(*SystemStatus)
		if !ok {
			return fmt.Errorf("invalid data type for status")
		}
		fmt.Printf("Service: %v, Devices: %d, Alerts: %d\n", status.Service.Running, len(status.Devices), len(status.Alerts))
	case "devices":
		devices, ok := data.([]string)
		if !ok {
			return fmt.Errorf("invalid data type for devices")
		}
		for _, d := range devices {
			fmt.Println(d)
		}
	case "alerts":
		alerts, ok := data.([]models.Alert)
		if !ok {
			return fmt.Errorf("invalid data type for alerts")
		}
		for _, a := range alerts {
			fmt.Printf("Alert: %s, Device: %s, Time: %s\n", a.Message, a.Device, a.Timestamp.Format(time.RFC3339))
		}
	case "predictions":
		// 类似 alerts 处理
	case "models":
		models, ok := data.([]string)
		if !ok {
			return fmt.Errorf("invalid data type for models")
		}
		for _, m := range models {
			fmt.Println(m)
		}
	}
	return nil
}
