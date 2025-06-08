package remediation

import (
	"github.com/turtacn/ioshelfer/internal/common/errors"
	"github.com/turtacn/ioshelfer/internal/common/logger"
	"github.com/turtacn/ioshelfer/internal/common/types/enum"
	"github.com/turtacn/ioshelfer/internal/core/detection"
	"go.uber.org/zap"
	"time"
)

// Config defines the configuration for the remediation engine.
type Config struct {
	AutoIsolation      bool          // Enable automatic isolation
	PreservePathsRatio float64       // Minimum ratio of healthy paths to preserve
	MinHealthyPaths    int           // Minimum number of healthy paths
	RecoveryTimeout    time.Duration // Timeout for recovery attempts
}

// RemediationResult represents the result of a remediation action.
type RemediationResult struct {
	DeviceType enum.DeviceType
	DeviceID   string
	Action     string // e.g., "isolated", "recovered"
	Success    bool
	Error      error
}

// Remediator defines the interface for remediation actions.
type Remediator interface {
	Isolate(deviceType enum.DeviceType, deviceID string, strategy enum.IsolationStrategy) (RemediationResult, error)
	Recover(deviceType enum.DeviceType, deviceID string) (RemediationResult, error)
}

// RAIDRemediator implements Remediator for RAID controllers.
type RAIDRemediator struct {
	config   *Config
	detector detection.Detector
}

// NewRAIDRemediator creates a new RAIDRemediator instance.
func NewRAIDRemediator(config *Config, detector detection.Detector) *RAIDRemediator {
	return &RAIDRemediator{
		config:   config,
		detector: detector,
	}
}

// Isolate isolates a RAID controller based on the specified strategy.
func (r *RAIDRemediator) Isolate(deviceType enum.DeviceType, deviceID string, strategy enum.IsolationStrategy) (RemediationResult, error) {
	if deviceType != enum.RAID {
		return RemediationResult{}, errors.New("invalid device type for RAID remediation", nil)
	}

	if !r.config.AutoIsolation {
		logger.Warn("auto-isolation disabled", zap.String("device_id", deviceID))
		return RemediationResult{
			DeviceType: deviceType,
			DeviceID:   deviceID,
			Action:     "isolation skipped",
			Success:    false,
		}, nil
	}

	// Check current health to ensure isolation is necessary
	status, err := r.detector.CheckSubHealth()
	if err != nil {
		return RemediationResult{}, errors.Wrap(err, "failed to check RAID health")
	}

	if status.Status == enum.Healthy {
		logger.Info("no isolation needed", zap.String("device_id", deviceID), zap.String("status", status.Status.String()))
		return RemediationResult{
			DeviceType: deviceType,
			DeviceID:   deviceID,
			Action:     "no action",
			Success:    true,
		}, nil
	}

	// Placeholder: Implement actual isolation logic (e.g., disable controller via system call)
	logger.Info("isolating RAID controller",
		zap.String("device_id", deviceID),
		zap.String("strategy", strategy.String()),
	)

	return RemediationResult{
		DeviceType: deviceType,
		DeviceID:   deviceID,
		Action:     "isolated",
		Success:    true,
	}, nil
}

// Recover attempts to recover a previously isolated RAID controller.
func (r *RAIDRemediator) Recover(deviceType enum.DeviceType, deviceID string) (RemediationResult, error) {
	if deviceType != enum.RAID {
		return RemediationResult{}, errors.New("invalid device type for RAID remediation", nil)
	}

	// Check current health to confirm recovery is feasible
	status, err := r.detector.CheckSubHealth()
	if err != nil {
		return RemediationResult{}, errors.Wrap(err, "failed to check RAID health")
	}

	if status.Status != enum.Healthy {
		logger.Warn("cannot recover unhealthy device",
			zap.String("device_id", deviceID),
			zap.String("status", status.Status.String()),
		)
		return RemediationResult{
			DeviceType: deviceType,
			DeviceID:   deviceID,
			Action:     "recovery failed",
			Success:    false,
			Error:      errors.New("device not healthy for recovery", nil),
		}, nil
	}

	// Placeholder: Implement actual recovery logic (e.g., re-enable controller)
	logger.Info("recovering RAID controller", zap.String("device_id", deviceID))

	return RemediationResult{
		DeviceType: deviceType,
		DeviceID:   deviceID,
		Action:     "recovered",
		Success:    true,
	}, nil
}
