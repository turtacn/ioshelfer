// pkg/network/traffic.go
package network

import (
	"time"
	"github.com/turtacn/ioshelfer/internal/common/errors"
	"github.com/turtacn/ioshelfer/internal/common/logger"
	"github.com/turtacn/ioshelfer/internal/infra/ebpf"
	"go.uber.org/zap"
)

// TrafficMetrics represents analyzed network traffic metrics.
type TrafficMetrics struct {
	Interface        string        `json:"interface"`
	TCPLatencyP50    time.Duration `json:"tcp_latency_p50"`
	TCPLatencyP95    time.Duration `json:"tcp_latency_p95"`
	UDPLatencyP50    time.Duration `json:"udp_latency_p50"`
	UDPLatencyP95    time.Duration `json:"udp_latency_p95"`
	PacketLossRate   float64       `json:"packet_loss_rate"`
	ThroughputMbps   float64       `json:"throughput_mbps"`
	Timestamp        time.Time     `json:"timestamp"`
}

// LatencyMetrics represents latency-specific metrics for monitoring.
type LatencyMetrics struct {
	Interface        string        `json:"interface"`
	TCPLatencyP95    time.Duration `json:"tcp_latency_p95"`
	UDPLatencyP95    time.Duration `json:"udp_latency_p95"`
	PacketLossRate   float64       `json:"packet_loss_rate"`
	LatencyTrend     string        `json:"latency_trend"` // "increasing", "stable", "decreasing"
	Timestamp        time.Time     `json:"timestamp"`
}

// TrafficAnalyzer defines the interface for network traffic analysis.
type TrafficAnalyzer interface {
	AnalyzeTraffic(interfaceName string) (*TrafficMetrics, error)
	MonitorLatency(interfaceName string, window time.Duration) (*LatencyMetrics, error)
}

// NetworkTrafficAnalyzer implements the TrafficAnalyzer interface.
type NetworkTrafficAnalyzer struct {
	ebpfMonitor *ebpf.Monitor
}

// NewNetworkTrafficAnalyzer creates a new NetworkTrafficAnalyzer instance.
func NewNetworkTrafficAnalyzer(ebpfMonitor *ebpf.Monitor) *NetworkTrafficAnalyzer {
	return &NetworkTrafficAnalyzer{
		ebpfMonitor: ebpfMonitor,
	}
}

// AnalyzeTraffic collects and analyzes network traffic metrics for a given interface.
func (a *NetworkTrafficAnalyzer) AnalyzeTraffic(interfaceName string) (*TrafficMetrics, error) {
	if interfaceName == "" {
		return nil, errors.New("empty interface name provided", nil)
	}

	// Fetch eBPF network metrics
	ebpfMetrics, err := a.ebpfMonitor.GetNetworkMetrics()
	if err != nil {
		return nil, errors.NewNetworkFailure("failed to collect network metrics", err)
	}

	// Simplified analysis: derive additional metrics from eBPF data
	metrics := &TrafficMetrics{
		Interface:      interfaceName,
		TCPLatencyP50:  ebpfMetrics.LatencyP95 / 2, // Simplified: assume P50 is half of P95
		TCPLatencyP95:  ebpfMetrics.LatencyP95,
		UDPLatencyP50:  ebpfMetrics.LatencyP95 / 2, // Simplified: similar assumption for UDP
		UDPLatencyP95:  ebpfMetrics.LatencyP95,
		PacketLossRate: ebpfMetrics.PacketLossRate,
		ThroughputMbps: float64(ebpfMetrics.BytesPerSecond) * 8 / 1_000_000, // Convert bytes/s to Mbps
		Timestamp:      time.Now(),
	}

	logger.Info("analyzed network traffic",
		zap.String("interface", interfaceName),
		zap.Duration("tcp_latency_p95", metrics.TCPLatencyP95),
		zap.Duration("udp_latency_p95", metrics.UDPLatencyP95),
		zap.Float64("packet_loss_rate", metrics.PacketLossRate),
		zap.Float64("throughput_mbps", metrics.ThroughputMbps),
	)

	return metrics, nil
}

// MonitorLatency monitors network latency trends over a time window.
func (a *NetworkTrafficAnalyzer) MonitorLatency(interfaceName string, window time.Duration) (*LatencyMetrics, error) {
	if interfaceName == "" {
		return nil, errors.New("empty interface name provided", nil)
	}

	// Fetch current eBPF metrics
	currentMetrics, err := a.ebpfMonitor.GetNetworkMetrics()
	if err != nil {
		return nil, errors.NewNetworkFailure("failed to collect network metrics for latency monitoring", err)
	}

	// Initialize latency metrics
	metrics := &LatencyMetrics{
		Interface:      interfaceName,
		TCPLatencyP95:  currentMetrics.LatencyP95,
		UDPLatencyP95:  currentMetrics.LatencyP95, // Simplified: assume UDP similar to TCP
		PacketLossRate: currentMetrics.PacketLossRate,
		Timestamp:      time.Now(),
	}

	// Analyze latency trend (simplified: compare current to a threshold-based trend)
	latencyTrend := "stable"
	if currentMetrics.LatencyP95 > 100*time.Millisecond {
		latencyTrend = "increasing"
	} else if currentMetrics.LatencyP95 < 50*time.Millisecond {
		latencyTrend = "decreasing"
	}
	metrics.LatencyTrend = latencyTrend

	// Log the monitoring results
	logger.Info("monitored network latency",
		zap.String("interface", interfaceName),
		zap.Duration("tcp_latency_p95", metrics.TCPLatencyP95),
		zap.Duration("udp_latency_p95", metrics.UDPLatencyP95),
		zap.Float64("packet_loss_rate", metrics.PacketLossRate),
		zap.String("latency_trend", metrics.LatencyTrend),
	)

	return metrics, nil
}
