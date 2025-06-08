package chaos

import (
	"context"
	"github.com/turtacn/ioshelfer/internal/common/errors"
	"github.com/turtacn/ioshelfer/internal/common/logger"
	"github.com/turtacn/ioshelfer/internal/common/types/enum"
	"go.uber.org/zap"
	"time"
)

// FaultType defines the type of fault to inject.
type FaultType string

const (
	NetworkLatency FaultType = "network_latency"
	DiskIOPS       FaultType = "disk_iops"
)

// ExperimentConfig defines the configuration for a chaos experiment.
type ExperimentConfig struct {
	DeviceType enum.DeviceType        // Target device type (e.g., Network, Disk)
	DeviceID   string                 // Target device ID (e.g., "eth0", "disk-0")
	FaultType  FaultType              // Type of fault to inject
	Duration   time.Duration          // Duration of the fault
	Parameters map[string]interface{} // Fault-specific parameters (e.g., latency in ms, IOPS limit)
}

// ExperimentResult represents the result of a chaos experiment.
type ExperimentResult struct {
	FaultType    FaultType
	DeviceType   enum.DeviceType
	DeviceID     string
	Success      bool
	Error        error
	Observations map[string]interface{} // Observed system metrics during experiment
}

// Experiment defines the interface for chaos engineering experiments.
type Experiment interface {
	Execute(ctx context.Context, config ExperimentConfig) (ExperimentResult, error)
	Validate(ctx context.Context, config ExperimentConfig) (ExperimentResult, error)
}

// ChaosExperiment implements the Experiment interface with mock fault injection.
type ChaosExperiment struct{}

// NewChaosExperiment creates a new ChaosExperiment instance.
func NewChaosExperiment() *ChaosExperiment {
	return &ChaosExperiment{}
}

// Execute injects a fault based on the provided configuration.
func (e *ChaosExperiment) Execute(ctx context.Context, config ExperimentConfig) (ExperimentResult, error) {
	result := ExperimentResult{
		FaultType:    config.FaultType,
		DeviceType:   config.DeviceType,
		DeviceID:     config.DeviceID,
		Observations: make(map[string]interface{}),
	}

	logger.Info("starting chaos experiment",
		zap.String("fault_type", string(config.FaultType)),
		zap.String("device_type", config.DeviceType.String()),
		zap.String("device_id", config.DeviceID),
		zap.Duration("duration", config.Duration),
	)

	// Validate context
	if ctx.Err() != nil {
		return result, errors.New("context cancelled before execution", ctx.Err())
	}

	switch config.FaultType {
	case NetworkLatency:
		if latency, ok := config.Parameters["latency_ms"].(float64); !ok || latency <= 0 {
			return result, errors.New("invalid or missing latency_ms parameter", nil)
		}
		// Placeholder: Simulate network latency injection (e.g., using tc netem)
		result.Observations["injected_latency_ms"] = config.Parameters["latency_ms"]
		result.Success = true
		logger.Info("injected network latency",
			zap.Float64("latency_ms", config.Parameters["latency_ms"].(float64)),
			zap.String("device_id", config.DeviceID),
		)

	case DiskIOPS:
		if iopsLimit, ok := config.Parameters["iops_limit"].(float64); !ok || iopsLimit <= 0 {
			return result, errors.New("invalid or missing iops_limit parameter", nil)
		}
		// Placeholder: Simulate disk IOPS restriction (e.g., using blkio cgroup)
		result.Observations["injected_iops_limit"] = config.Parameters["iops_limit"]
		result.Success = true
		logger.Info("injected disk IOPS limit",
			zap.Float64("iops_limit", config.Parameters["iops_limit"].(float64)),
			zap.String("device_id", config.DeviceID),
		)

	default:
		return result, errors.New("unsupported fault type", nil)
	}

	// Simulate fault duration
	select {
	case <-time.After(config.Duration):
		logger.Info("chaos experiment completed",
			zap.String("fault_type", string(config.FaultType)),
			zap.String("device_id", config.DeviceID),
		)
	case <-ctx.Done():
		result.Error = ctx.Err()
		result.Success = false
		logger.Warn("chaos experiment interrupted",
			zap.String("fault_type", string(config.FaultType)),
			zap.String("device_id", config.DeviceID),
			zap.Error(ctx.Err()),
		)
	}

	return result, nil
}

// Validate checks system resilience after fault injection.
func (e *ChaosExperiment) Validate(ctx context.Context, config ExperimentConfig) (ExperimentResult, error) {
	result := ExperimentResult{
		FaultType:    config.FaultType,
		DeviceType:   config.DeviceType,
		DeviceID:     config.DeviceID,
		Observations: make(map[string]interface{}),
	}

	logger.Info("validating system resilience",
		zap.String("fault_type", string(config.FaultType)),
		zap.String("device_type", config.DeviceType.String()),
		zap.String("device_id", config.DeviceID),
	)

	if ctx.Err() != nil {
		return result, errors.New("context cancelled before validation", ctx.Err())
	}

	// Placeholder: Validate system behavior (e.g., check if system recovered or maintained SLA)
	switch config.FaultType {
	case NetworkLatency:
		// Simulate validation: Check if latency-affected services recovered
		result.Observations["recovery_status"] = "recovered"
		result.Success = true
		logger.Info("validated network latency recovery",
			zap.String("device_id", config.DeviceID),
		)

	case DiskIOPS:
		// Simulate validation: Check if IOPS-limited disk still meets minimum performance
		result.Observations["recovery_status"] = "recovered"
		result.Success = true
		logger.Info("validated disk IOPS recovery",
			zap.String("device_id", config.DeviceID),
		)

	default:
		return result, errors.New("unsupported fault type for validation", nil)
	}

	return result, nil
}
