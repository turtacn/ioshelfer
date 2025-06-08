package detection

import (
	"time"
	"github.com/turtacn/ioshelfer/internal/common/errors"
	"github.com/turtacn/ioshelfer/internal/common/types/enum"
	"github.com/turtacn/ioshelfer/internal/infra/ebpf"
)

// Config defines the configuration for the detection engine.
type Config struct {
	QueueThreshold    int           // Maximum queue depth before sub-health is detected
	LatencyThreshold  time.Duration // Maximum I/O latency before sub-health is detected
	MonitorInterval   time.Duration // Interval for periodic health checks
	IOPSVarThreshold  float64       // IOPS variance threshold for disk sub-health
	PacketLossThreshold float64     // Packet loss rate threshold for network sub-health
}

// HealthStatus represents the result of a sub-health check.
type HealthStatus struct {
	DeviceType      enum.DeviceType
	Status          enum.HealthStatus
	Confidence      float64
	Recommendation  string
	Metrics         interface{} // Device-specific metrics (RAID, Disk, or Network)
}

// Detector defines the interface for sub-health detection.
type Detector interface {
	CheckSubHealth() (HealthStatus, error)
}

// RAIDDetector implements Detector for RAID controllers.
type RAIDDetector struct {
	config     *Config
	ebpfMonitor *ebpf.Monitor
}

// NewRAIDDetector creates a new RAIDDetector instance.
func NewRAIDDetector(config *Config, monitor *ebpf.Monitor) *RAIDDetector {
	return &RAIDDetector{
		config:     config,
		ebpfMonitor: monitor,
	}
}

// CheckSubHealth performs a sub-health check for a RAID controller.
func (d *RAIDDetector) CheckSubHealth() (HealthStatus, error) {
	metrics, err := d.ebpfMonitor.GetRAIDMetrics()
	if err != nil {
		return HealthStatus{}, errors.NewQueueOverflow("failed to get RAID metrics", err)
	}

	status := enum.Healthy
	confidence := 1.0
	recommendation := "no action required"

	if metrics.QueueDepth >= d.config.QueueThreshold {
		status = enum.SubHealthy
		confidence = 0.95
		recommendation = "temporary isolation recommended"
	}

	if metrics.AvgLatency > d.config.LatencyThreshold {
		status = enum.SubHealthy
		confidence = 0.90
		recommendation = "check controller firmware and isolate if persistent"
	}

	if metrics.ErrorRetryRate > 100 { // Example threshold: 100 retries/hour
		status = enum.Failed
		confidence = 0.99
		recommendation = "immediate isolation and replacement"
	}

	return HealthStatus{
		DeviceType:     enum.RAID,
		Status:         status,
		Confidence:     confidence,
		Recommendation: recommendation,
		Metrics:        metrics,
	}, nil
}

// DiskDetector implements Detector for disk devices.
type DiskDetector struct {
	config     *Config
	ebpfMonitor *ebpf.Monitor
}

// NewDiskDetector creates a new DiskDetector instance.
func NewDiskDetector(config *Config, monitor *ebpf.Monitor) *DiskDetector {
	return &DiskDetector{
		config:     config,
		ebpfMonitor: monitor,
	}
}

// CheckSubHealth performs a sub-health check for a disk device.
func (d *DiskDetector) CheckSubHealth() (HealthStatus, error) {
	metrics, err := d.ebpfMonitor.GetDiskMetrics()
	if err != nil {
		return HealthStatus{}, errors.NewStorageFailure("failed to get disk metrics", err)
	}

	status := enum.Healthy
	confidence := 1.0
	recommendation := "no action required"

	if metrics.IOPSVariance > d.config.IOPSVarThreshold {
		status = enum.SubHealthy
		confidence = 0.92
		recommendation = "monitor disk performance closely"
	}

	if metrics.SMART.ReallocatedSectors > 100 { // Example threshold
		status = enum.SubHealthy
		confidence = 0.95
		recommendation = "schedule disk replacement"
	}

	return HealthStatus{
		DeviceType:     enum.Disk,
		Status:         status,
		Confidence:     confidence,
		Recommendation: recommendation,
		Metrics:        metrics,
	}, nil
}

// NetworkDetector implements Detector for network I/O.
type NetworkDetector struct {
	config     *Config
	ebpfMonitor *ebpf.Monitor
}

// NewNetworkDetector creates a new NetworkDetector instance.
func NewNetworkDetector(config *Config, monitor *ebpf.Monitor) *NetworkDetector {
	return &NetworkDetector{
		config:     config,
		ebpfMonitor: monitor,
	}
}

// CheckSubHealth performs a sub-health check for network I/O.
func (d *NetworkDetector) CheckSubHealth() (HealthStatus, error) {
	metrics, err := d.ebpfMonitor.GetNetworkMetrics()
	if err != nil {
		return HealthStatus{}, errors.NewNetworkPacketLoss("failed to get network metrics", err)
	}

	status := enum.Healthy
	confidence := 1.0
	recommendation := "no action required"

	if metrics.PacketLossRate > d.config.PacketLossThreshold {
		status = enum.SubHealthy
		confidence = 0.93
		recommendation = "check network interface and routing"
	}

	if metrics.LatencyP95 > d.config.LatencyThreshold {
		status = enum.SubHealthy
		confidence = 0.90
		recommendation = "investigate network congestion"
	}

	return HealthStatus{
		DeviceType:     enum.Network,
		Status:         status,
		Confidence:     confidence,
		Recommendation: recommendation,
		Metrics:        metrics,
	}, nil
}