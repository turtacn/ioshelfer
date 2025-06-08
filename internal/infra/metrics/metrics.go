package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/turtacn/ioshelfer/internal/common/logger"
	"github.com/turtacn/ioshelfer/internal/common/types/enum"
	"github.com/turtacn/ioshelfer/internal/core/detection"
	"github.com/turtacn/ioshelfer/internal/infra/ebpf"
	"go.uber.org/zap"
)

// MetricsCollector defines the interface for collecting and exposing metrics.
type MetricsCollector interface {
	Collect(deviceType enum.DeviceType, status detection.HealthStatus)
	ServeHTTP(port string) error
}

// PrometheusCollector implements MetricsCollector using Prometheus.
type PrometheusCollector struct {
	raidQueueDepth     *prometheus.GaugeVec
	diskIOPSVariance   *prometheus.GaugeVec
	networkLatencyP95  *prometheus.GaugeVec
	detector           detection.Detector
}

// NewPrometheusCollector creates a new PrometheusCollector instance.
func NewPrometheusCollector(detector detection.Detector) *PrometheusCollector {
	collector := &PrometheusCollector{
		raidQueueDepth: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ioshelfer_raid_queue_depth",
				Help: "Current queue depth of RAID controllers",
			},
			[]string{"controller_id"},
		),
		diskIOPSVariance: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ioshelfer_disk_iops_variance",
				Help: "IOPS variance for disk devices",
			},
			[]string{"disk_id"},
		),
		networkLatencyP95: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ioshelfer_network_latency_p95_seconds",
				Help: "95th percentile latency for network I/O",
			},
			[]string{"interface"},
		),
		detector: detector,
	}

	// Register metrics with Prometheus
	prometheus.MustRegister(collector.raidQueueDepth)
	prometheus.MustRegister(collector.diskIOPSVariance)
	prometheus.MustRegister(collector.networkLatencyP95)
	return collector
}

// Collect gathers metrics from the detection engine and updates Prometheus gauges.
func (c *PrometheusCollector) Collect(deviceType enum.DeviceType, status detection.HealthStatus) {
	switch deviceType {
	case enum.RAID:
		if metrics, ok := status.Metrics.(*ebpf.RAIDMetrics); ok {
			c.raidQueueDepth.WithLabelValues("raid-0").Set(float64(metrics.QueueDepth))
			logger.Info("collected RAID queue depth",
				zap.Int("queue_depth", metrics.QueueDepth),
				zap.String("controller_id", "raid-0"),
			)
		}
	case enum.Disk:
		if metrics, ok := status.Metrics.(*ebpf.DiskMetrics); ok {
			c.diskIOPSVariance.WithLabelValues("disk-0").Set(metrics.IOPSVariance)
			logger.Info("collected disk IOPS variance",
				zap.Float64("iops_variance", metrics.IOPSVariance),
				zap.String("disk_id", "disk-0"),
			)
		}
	case enum.Network:
		if metrics, ok := status.Metrics.(*ebpf.NetworkMetrics); ok {
			c.networkLatencyP95.WithLabelValues("eth0").Set(float64(metrics.LatencyP95.Seconds()))
			logger.Info("collected network latency",
				zap.Float64("latency_p95_seconds", metrics.LatencyP95.Seconds()),
				zap.String("interface", "eth0"),
			)
		}
	}
}