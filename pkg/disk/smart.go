// pkg/disk/smart.go
package disk

import (
	"strconv"
	"strings"
	"time"
	"github.com/turtacn/ioshelfer/internal/common/errors"
	"github.com/turtacn/ioshelfer/internal/common/logger"
	"github.com/turtacn/ioshelfer/internal/common/types/enum"
	"github.com/turtacn/ioshelfer/internal/infra/storage"
	"go.uber.org/zap"
)

// SMARTAttribute represents a single SMART attribute.
type SMARTAttribute struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Value       int    `json:"value"`
	Worst       int    `json:"worst"`
	Threshold   int    `json:"threshold"`
	RawValue    int64  `json:"raw_value"`
	Status      string `json:"status"`
}

// SMARTData represents the complete SMART data for a disk.
type SMARTData struct {
	DeviceID           string                     `json:"device_id"`
	Model              string                     `json:"model"`
	SerialNumber       string                     `json:"serial_number"`
	Attributes         map[int]SMARTAttribute     `json:"attributes"`
	OverallStatus      enum.HealthStatus          `json:"overall_status"`
	ReallocatedSectors int                        `json:"reallocated_sectors"`
	ReadErrorRate      float64                    `json:"read_error_rate"`
	Temperature        int                        `json:"temperature"`
	PowerOnHours       int64                      `json:"power_on_hours"`
	Timestamp          time.Time                  `json:"timestamp"`
}

// PerformanceMetrics represents disk performance metrics over time.
type PerformanceMetrics struct {
	DeviceID        string    `json:"device_id"`
	IOPSVariance    float64   `json:"iops_variance"`
	LatencyTrend    string    `json:"latency_trend"` // "increasing", "stable", "decreasing"
	ErrorTrend      string    `json:"error_trend"`   // "increasing", "stable", "decreasing"
	PredictedFailure bool     `json:"predicted_failure"`
	Timestamp       time.Time `json:"timestamp"`
}

// Monitor defines the interface for SMART data monitoring.
type Monitor interface {
	ParseSMART(rawData string) (*SMARTData, error)
	MonitorPerformance(deviceID string, window time.Duration) (*PerformanceMetrics, error)
	GetHistoricalData(deviceID string, window time.Duration) ([]SMARTData, error)
}

// SMARTMonitor implements the Monitor interface.
type SMARTMonitor struct {
	storage storage.Storage
}

// NewSMARTMonitor creates a new SMARTMonitor instance.
func NewSMARTMonitor(storage storage.Storage) *SMARTMonitor {
	return &SMARTMonitor{
		storage: storage,
	}
}

// ParseSMART parses raw SMART data output and returns structured SMART data.
func (m *SMARTMonitor) ParseSMART(rawData string) (*SMARTData, error) {
	if rawData == "" {
		return nil, errors.New("empty SMART data provided", nil)
	}

	smartData := &SMARTData{
		Attributes: make(map[int]SMARTAttribute),
		Timestamp:  time.Now(),
	}

	lines := strings.Split(rawData, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Parse device information
		if strings.HasPrefix(line, "Device Model:") {
			smartData.Model = strings.TrimSpace(strings.Split(line, ":")[1])
		} else if strings.HasPrefix(line, "Serial Number:") {
			smartData.SerialNumber = strings.TrimSpace(strings.Split(line, ":")[1])
		} else if strings.HasPrefix(line, "Device:") {
			smartData.DeviceID = strings.TrimSpace(strings.Split(line, ":")[1])
		}

		// Parse SMART attributes (simplified format)
		if strings.Contains(line, "Reallocated_Sector_Ct") {
			if attr, err := m.parseAttributeLine(line, 5); err == nil {
				smartData.Attributes[5] = attr
				smartData.ReallocatedSectors = int(attr.RawValue)
			}
		} else if strings.Contains(line, "Read_Error_Rate") {
			if attr, err := m.parseAttributeLine(line, 1); err == nil {
				smartData.Attributes[1] = attr
				// Convert raw value to error rate percentage
				smartData.ReadErrorRate = float64(attr.RawValue) / 1000000.0
			}
		} else if strings.Contains(line, "Temperature_Celsius") {
			if attr, err := m.parseAttributeLine(line, 194); err == nil {
				smartData.Attributes[194] = attr
				smartData.Temperature = int(attr.RawValue)
			}
		} else if strings.Contains(line, "Power_On_Hours") {
			if attr, err := m.parseAttributeLine(line, 9); err == nil {
				smartData.Attributes[9] = attr
				smartData.PowerOnHours = attr.RawValue
			}
		}
	}

	// Determine overall health status
	smartData.OverallStatus = m.assessOverallHealth(smartData)

	// Store the parsed data
	metric := storage.Metric{
		Timestamp:  smartData.Timestamp,
		DeviceType: enum.Disk,
		DeviceID:   smartData.DeviceID,
		Value:      smartData,
	}
	if err := m.storage.Store(metric); err != nil {
		logger.Warn("failed to store SMART data", zap.Error(err))
	}

	logger.Info("parsed SMART data",
		zap.String("device_id", smartData.DeviceID),
		zap.String("model", smartData.Model),
		zap.String("status", smartData.OverallStatus.String()),
		zap.Int("reallocated_sectors", smartData.ReallocatedSectors),
		zap.Float64("read_error_rate", smartData.ReadErrorRate),
	)

	return smartData, nil
}

// parseAttributeLine parses a single SMART attribute line.
func (m *SMARTMonitor) parseAttributeLine(line string, expectedID int) (SMARTAttribute, error) {
	fields := strings.Fields(line)
	if len(fields) < 6 {
		return SMARTAttribute{}, errors.New("invalid SMART attribute line format", nil)
	}

	id, err := strconv.Atoi(fields[0])
	if err != nil {
		return SMARTAttribute{}, errors.Wrap(err, "failed to parse attribute ID")
	}

	value, err := strconv.Atoi(fields[2])
	if err != nil {
		return SMARTAttribute{}, errors.Wrap(err, "failed to parse attribute value")
	}

	worst, err := strconv.Atoi(fields[3])
	if err != nil {
		return SMARTAttribute{}, errors.Wrap(err, "failed to parse worst value")
	}

	threshold, err := strconv.Atoi(fields[4])
	if err != nil {
		return SMARTAttribute{}, errors.Wrap(err, "failed to parse threshold")
	}

	rawValue, err := strconv.ParseInt(fields[len(fields)-1], 10, 64)
	if err != nil {
		return SMARTAttribute{}, errors.Wrap(err, "failed to parse raw value")
	}

	status := "OK"
	if value <= threshold {
		status = "FAILING"
	}

	return SMARTAttribute{
		ID:        id,
		Name:      fields[1],
		Value:     value,
		Worst:     worst,
		Threshold: threshold,
		RawValue:  rawValue,
		Status:    status,
	}, nil
}

// assessOverallHealth determines the overall health status based on SMART attributes.
func (m *SMARTMonitor) assessOverallHealth(data *SMARTData) enum.HealthStatus {
	// Critical thresholds
	if data.ReallocatedSectors > 100 {
		return enum.Failed
	}
	if data.ReadErrorRate > 0.1 { // 0.1% error rate
		return enum.Failed
	}
	if data.Temperature > 65 { // 65°C
		return enum.SubHealthy
	}

	// Check for any failing attributes
	for _, attr := range data.Attributes {
		if attr.Status == "FAILING" {
			return enum.SubHealthy
		}
	}

	// Warning thresholds
	if data.ReallocatedSectors > 10 {
		return enum.SubHealthy
	}
	if data.ReadErrorRate > 0.01 { // 0.01% error rate
		return enum.SubHealthy
	}

	return enum.Healthy
}

// MonitorPerformance analyzes disk performance trends over time.
func (m *SMARTMonitor) MonitorPerformance(deviceID string, window time.Duration) (*PerformanceMetrics, error) {
	// Get historical data
	metrics, err := m.storage.Query(enum.Disk, deviceID, window)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query historical SMART data")
	}

	if len(metrics) < 2 {
		return &PerformanceMetrics{
			DeviceID:        deviceID,
			IOPSVariance:    0.0,
			LatencyTrend:    "stable",
			ErrorTrend:      "stable",
			PredictedFailure: false,
			Timestamp:       time.Now(),
		}, nil
	}

	// Analyze trends (simplified implementation)
	performance := &PerformanceMetrics{
		DeviceID:     deviceID,
		IOPSVariance: m.calculateIOPSVariance(metrics),
		Timestamp:    time.Now(),
	}

	// Analyze error trend
	performance.ErrorTrend = m.analyzeErrorTrend(metrics)

	// Predict failure based on trends
	performance.PredictedFailure = m.predictFailure(metrics)

	logger.Info("monitored disk performance",
		zap.String("device_id", deviceID),
		zap.Float64("iops_variance", performance.IOPSVariance),
		zap.String("error_trend", performance.ErrorTrend),
		zap.Bool("predicted_failure", performance.PredictedFailure),
	)

	return performance, nil
}

// calculateIOPSVariance calculates the variance in IOPS over historical data.
func (m *SMARTMonitor) calculateIOPSVariance(metrics []storage.Metric) float64 {
	// Simplified calculation - in reality would use actual IOPS data
	// Here we use reallocated sectors as a proxy for disk health variance
	if len(metrics) < 2 {
		return 0.0
	}

	var values []float64
	for _, metric := range metrics {
		if smartData, ok := metric.Value.(*SMARTData); ok {
			values = append(values, float64(smartData.ReallocatedSectors))
		}
	}

	if len(values) < 2 {
		return 0.0
	}

	// Calculate variance
	mean := 0.0
	for _, v := range values {
		mean += v
	}
	mean /= float64(len(values))

	variance := 0.0
	for _, v := range values {
		variance += (v - mean) * (v - mean)
	}
	variance /= float64(len(values))

	return variance
}

// analyzeErrorTrend analyzes the trend of error rates over time.
func (m *SMARTMonitor) analyzeErrorTrend(metrics []storage.Metric) string {
	if len(metrics) < 2 {
		return "stable"
	}

	var errorRates []float64
	for _, metric := range metrics {
		if smartData, ok := metric.Value.(*SMARTData); ok {
			errorRates = append(errorRates, smartData.ReadErrorRate)
		}
	}

	if len(errorRates) < 2 {
		return "stable"
	}

	// Simple trend analysis
	first := errorRates[0]
	last := errorRates[len(errorRates)-1]

	if last > first*1.5 {
		return "increasing"
	} else if last < first*0.5 {
		return "decreasing"
	}
	return "stable"
}

// predictFailure predicts potential disk failure based on historical trends.
func (m *SMARTMonitor) predictFailure(metrics []storage.Metric) bool {
	if len(metrics) < 3 {
		return false
	}

	// Check for rapid degradation
	latest := metrics[len(metrics)-1]
	if smartData, ok := latest.Value.(*SMARTData); ok {
		// Predict failure if reallocated sectors > 50 or read error rate > 0.05%
		if smartData.ReallocatedSectors > 50 || smartData.ReadErrorRate > 0.05 {
			return true
		}
	}

	return false
}

// GetHistoricalData retrieves historical SMART data for a device.
func (m *SMARTMonitor) GetHistoricalData(deviceID string, window time.Duration) ([]SMARTData, error) {
	metrics, err := m.storage.Query(enum.Disk, deviceID, window)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query historical data")
	}

	var smartDataList []SMARTData
	for _, metric := range metrics {
		if smartData, ok := metric.Value.(*SMARTData); ok {
			smartDataList = append(smartDataList, *smartData)
		}
	}

	return smartDataList, nil
}
