// internal/core/slo/slo.go
package slo

import (
	"fmt"
	"time"
	"github.com/turtacn/ioshelfer/internal/common/logger"
	"github.com/turtacn/ioshelfer/internal/common/types/enum"
	"go.uber.org/zap"
)

// ServiceTier defines the criticality level of a business service.
type ServiceTier string

const (
	Critical    ServiceTier = "critical"
	NonCritical ServiceTier = "non_critical"
)

// SLOConfig defines the SLO requirements for different device types and service tiers.
type SLOConfig struct {
	ServiceTier        ServiceTier
	MaxLatency         time.Duration // Maximum acceptable I/O latency
	MinThroughput      float64       // Minimum acceptable throughput (MB/s)
	MaxThroughputLoss  float64       // Maximum acceptable throughput loss percentage
	MinAvailability    float64       // Minimum acceptable availability percentage
}

// SLIMetrics represents the current Service Level Indicators.
type SLIMetrics struct {
	DeviceType     enum.DeviceType
	DeviceID       string
	Latency        time.Duration
	Throughput     float64 // Current throughput in MB/s
	ThroughputLoss float64 // Throughput loss percentage
	Availability   float64 // Availability percentage
	Timestamp      time.Time
}

// ValidationResult represents the result of SLO validation.
type ValidationResult struct {
	DeviceType     enum.DeviceType
	DeviceID       string
	ServiceTier    ServiceTier
	SLOStatus      enum.HealthStatus
	Violations     []string
	Confidence     float64
	Recommendation string
}

// Validator defines the interface for SLO validation.
type Validator interface {
	ValidateSLO(config SLOConfig, metrics SLIMetrics) (ValidationResult, error)
	GetSLOConfig(deviceType enum.DeviceType, serviceTier ServiceTier) SLOConfig
}

// SLOValidator implements the Validator interface.
type SLOValidator struct {
	configs map[string]SLOConfig // Key: deviceType_serviceTier
}

// NewSLOValidator creates a new SLOValidator with default configurations.
func NewSLOValidator() *SLOValidator {
	validator := &SLOValidator{
		configs: make(map[string]SLOConfig),
	}
	validator.initDefaultConfigs()
	return validator
}

// initDefaultConfigs initializes default SLO configurations for different device types and service tiers.
func (v *SLOValidator) initDefaultConfigs() {
	// Critical RAID SLO
	v.configs["RAID_critical"] = SLOConfig{
		ServiceTier:       Critical,
		MaxLatency:        50 * time.Millisecond,
		MinThroughput:     500.0, // 500 MB/s
		MaxThroughputLoss: 10.0,  // 10%
		MinAvailability:   99.9,  // 99.9%
	}

	// Non-critical RAID SLO
	v.configs["RAID_non_critical"] = SLOConfig{
		ServiceTier:       NonCritical,
		MaxLatency:        100 * time.Millisecond,
		MinThroughput:     200.0, // 200 MB/s
		MaxThroughputLoss: 20.0,  // 20%
		MinAvailability:   99.0,  // 99.0%
	}

	// Critical Disk SLO
	v.configs["Disk_critical"] = SLOConfig{
		ServiceTier:       Critical,
		MaxLatency:        30 * time.Millisecond,
		MinThroughput:     100.0, // 100 MB/s
		MaxThroughputLoss: 5.0,   // 5%
		MinAvailability:   99.95, // 99.95%
	}

	// Non-critical Disk SLO
	v.configs["Disk_non_critical"] = SLOConfig{
		ServiceTier:       NonCritical,
		MaxLatency:        100 * time.Millisecond,
		MinThroughput:     50.0, // 50 MB/s
		MaxThroughputLoss: 15.0, // 15%
		MinAvailability:   98.0, // 98.0%
	}

	// Critical Network SLO
	v.configs["Network_critical"] = SLOConfig{
		ServiceTier:       Critical,
		MaxLatency:        10 * time.Millisecond,
		MinThroughput:     1000.0, // 1 GB/s
		MaxThroughputLoss: 5.0,    // 5%
		MinAvailability:   99.99,  // 99.99%
	}

	// Non-critical Network SLO
	v.configs["Network_non_critical"] = SLOConfig{
		ServiceTier:       NonCritical,
		MaxLatency:        50 * time.Millisecond,
		MinThroughput:     100.0, // 100 MB/s
		MaxThroughputLoss: 20.0,  // 20%
		MinAvailability:   99.0,  // 99.0%
	}
}

// GetSLOConfig retrieves the SLO configuration for a specific device type and service tier.
func (v *SLOValidator) GetSLOConfig(deviceType enum.DeviceType, serviceTier ServiceTier) SLOConfig {
	key := deviceType.String() + "_" + string(serviceTier)
	if config, exists := v.configs[key]; exists {
		return config
	}
	// Return default non-critical config if specific config not found
	return SLOConfig{
		ServiceTier:       NonCritical,
		MaxLatency:        100 * time.Millisecond,
		MinThroughput:     50.0,
		MaxThroughputLoss: 20.0,
		MinAvailability:   95.0,
	}
}

// ValidateSLO validates whether the current SLI metrics meet the SLO requirements.
func (v *SLOValidator) ValidateSLO(config SLOConfig, metrics SLIMetrics) (ValidationResult, error) {
	result := ValidationResult{
		DeviceType:  metrics.DeviceType,
		DeviceID:    metrics.DeviceID,
		ServiceTier: config.ServiceTier,
		SLOStatus:   enum.Healthy,
		Violations:  []string{},
		Confidence:  1.0,
	}

	// Validate latency
	if metrics.Latency > config.MaxLatency {
		result.SLOStatus = enum.SubHealthy
		result.Violations = append(result.Violations,
			fmt.Sprintf("latency violation: %v > %v", metrics.Latency, config.MaxLatency))
		result.Confidence = min(result.Confidence, 0.9)
	}

	// Validate throughput
	if metrics.Throughput < config.MinThroughput {
		result.SLOStatus = enum.SubHealthy
		result.Violations = append(result.Violations,
			fmt.Sprintf("throughput violation: %.2f < %.2f MB/s", metrics.Throughput, config.MinThroughput))
		result.Confidence = min(result.Confidence, 0.85)
	}

	// Validate throughput loss
	if metrics.ThroughputLoss > config.MaxThroughputLoss {
		result.SLOStatus = enum.SubHealthy
		result.Violations = append(result.Violations,
			fmt.Sprintf("throughput loss violation: %.2f%% > %.2f%%", metrics.ThroughputLoss, config.MaxThroughputLoss))
		result.Confidence = min(result.Confidence, 0.8)
	}

	// Validate availability
	if metrics.Availability < config.MinAvailability {
		if metrics.Availability < config.MinAvailability-5.0 { // More than 5% below threshold
			result.SLOStatus = enum.Failed
		} else {
			result.SLOStatus = enum.SubHealthy
		}
		result.Violations = append(result.Violations,
			fmt.Sprintf("availability violation: %.2f%% < %.2f%%", metrics.Availability, config.MinAvailability))
		result.Confidence = min(result.Confidence, 0.7)
	}

	// Generate recommendation based on violations
	if len(result.Violations) == 0 {
		result.Recommendation = "no action required"
	} else if result.SLOStatus == enum.Failed {
		result.Recommendation = "immediate intervention required"
	} else {
		result.Recommendation = "monitor closely and consider optimization"
	}

	logger.Info("SLO validation completed",
		zap.String("device_type", metrics.DeviceType.String()),
		zap.String("device_id", metrics.DeviceID),
		zap.String("service_tier", string(config.ServiceTier)),
		zap.String("slo_status", result.SLOStatus.String()),
		zap.Int("violations", len(result.Violations)),
		zap.Float64("confidence", result.Confidence),
	)

	return result, nil
}

// min returns the minimum of two float64 values.
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
