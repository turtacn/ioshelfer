// pkg/raid/controller.go
package raid

import (
	"time"
	"github.com/turtacn/ioshelfer/internal/common/errors"
	"github.com/turtacn/ioshelfer/internal/common/logger"
	"github.com/turtacn/ioshelfer/internal/common/types/enum"
	"github.com/turtacn/ioshelfer/internal/infra/ebpf"
	"go.uber.org/zap"
)

// Config defines the configuration for the RAID controller monitor.
type Config struct {
	QueueThreshold     int           // Maximum queue depth before sub-health is detected
	LatencyThreshold   time.Duration // Maximum I/O latency before sub-health is detected
	FirmwareVersion    string        // Expected firmware version
	MonitorInterval    time.Duration // Interval for periodic metric collection
}

// HealthStatus represents the health status of a RAID controller.
type HealthStatus struct {
	ControllerID      string
	Status            enum.HealthStatus
	QueueDepth        int
	AvgLatency        time.Duration
	FirmwareStatus    string
	Confidence        float64
	Recommendation    string
}

// Controller defines the interface for RAID controller monitoring.
type Controller interface {
	CheckHealth(controllerID string) (HealthStatus, error)
	GetMetrics(controllerID string) (*ebpf.RAIDMetrics, error)
}

// RAIDController implements the Controller interface.
type RAIDController struct {
	config  *Config
	monitor *ebpf.Monitor
}

// NewRAIDController creates a new RAIDController instance.
func NewRAIDController(config *Config, monitor *ebpf.Monitor) *RAIDController {
	return &RAIDController{
		config:  config,
		monitor: monitor,
	}
}

// CheckHealth evaluates the health of a RAID controller based on collected metrics.
func (c *RAIDController) CheckHealth(controllerID string) (HealthStatus, error) {
	metrics, err := c.GetMetrics(controllerID)
	if err != nil {
		return HealthStatus{}, errors.Wrap(err, "failed to check RAID controller health")
	}

	status := enum.Healthy
	confidence := 1.0
	recommendation := "no action required"
	firmwareStatus := "matched"

	// Check queue depth
	if metrics.QueueDepth >= c.config.QueueThreshold {
		status = enum.SubHealthy
		confidence = 0.95
		recommendation = "temporary isolation recommended"
	}

	// Check latency
	if metrics.AvgLatency > c.config.LatencyThreshold {
		status = enum.SubHealthy
		confidence = min(confidence, 0.90)
		recommendation = "check controller firmware and isolate if persistent"
	}

	// Check error retry rate
	if metrics.ErrorRetryRate > 100 { // Example threshold: 100 retries/hour
		status = enum.Failed
		confidence = 0.99
		recommendation = "immediate isolation and replacement"
	}

	// Check firmware (mocked check, assumes external validation)
	if c.config.FirmwareVersion != "" && c.config.FirmwareVersion != "v1.0" { // Mock firmware version
		firmwareStatus = "mismatch"
		status = enum.SubHealthy
		confidence = min(confidence, 0.85)
		recommendation = "update firmware to " + c.config.FirmwareVersion
		logger.Warn("firmware mismatch detected",
			zap.String("controller_id", controllerID),
			zap.String("expected_version", c.config.FirmwareVersion),
			zap.String("actual_version", "v1.0"),
		)
	}

	logger.Info("RAID controller health checked",
		zap.String("controller_id", controllerID),
		zap.String("status", status.String()),
		zap.Float64("confidence", confidence),
		zap.String("recommendation", recommendation),
	)

	return HealthStatus{
		ControllerID:   controllerID,
		Status:         status,
		QueueDepth:     metrics.QueueDepth,
		AvgLatency:     metrics.AvgLatency,
		FirmwareStatus: firmwareStatus,
		Confidence:     confidence,
		Recommendation: recommendation,
	}, nil
}

// GetMetrics retrieves RAID controller metrics using eBPF.
func (c *RAIDController) GetMetrics(controllerID string) (*ebpf.RAIDMetrics, error) {
	metrics, err := c.monitor.GetRAIDMetrics()
	if err != nil {
		return nil, errors.NewQueueOverflow("failed to get RAID metrics", err)
	}

	logger.Info("collected RAID metrics",
		zap.String("controller_id", controllerID),
		zap.Int("queue_depth", metrics.QueueDepth),
		zap.Duration("avg_latency", metrics.AvgLatency),
		zap.Int("error_retry_rate", metrics.ErrorRetryRate),
	)
	return metrics, nil
}

// min returns the minimum of two float64 values.
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
