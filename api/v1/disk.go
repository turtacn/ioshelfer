// api/v1/disk.go
package v1

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"
	"github.com/turtacn/ioshelfer/internal/common/errors"
	"github.com/turtacn/ioshelfer/internal/common/logger"
	"github.com/turtacn/ioshelfer/internal/common/types/enum"
	"github.com/turtacn/ioshelfer/internal/core/prediction"
	"github.com/turtacn/ioshelfer/pkg/disk"
	"go.uber.org/zap"
)

// DiskHandler handles disk-related API requests.
type DiskHandler struct {
	predictor *prediction.Predictor
}

// NewDiskHandler creates a new DiskHandler instance.
func NewDiskHandler(predictor *prediction.Predictor) *DiskHandler {
	return &DiskHandler{
		predictor: predictor,
	}
}

// RegisterRoutes registers disk-related API routes.
func (h *DiskHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/disks/predictions", h.handleDiskPredictions)
	mux.HandleFunc("/api/v1/disks/metrics", h.handleDiskMetrics)
}

// handleDiskPredictions handles requests to /api/v1/disks/predictions.
func (h *DiskHandler) handleDiskPredictions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract device ID and time window from query parameters
	deviceID := r.URL.Query().Get("device_id")
	if deviceID == "" {
		http.Error(w, "device_id is required", http.StatusBadRequest)
		return
	}

	windowStr := r.URL.Query().Get("window")
	window, err := time.ParseDuration(windowStr)
	if err != nil || window <= 0 {
		window = 24 * time.Hour // Default to 24 hours
	}

	// Get prediction results
	smartData, metrics, err := h.predictor.PredictDiskFailure(deviceID, window)
	if err != nil {
		logger.Error("failed to predict disk failure",
			zap.String("device_id", deviceID),
			zap.Duration("window", window),
			zap.Error(err))
		if errors.Is(err, errors.NewNotFound("", nil)) {
			http.Error(w, "disk not found", http.StatusNotFound)
		} else {
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
		return
	}

	// Prepare JSON response
	response := struct {
		DeviceID           string            `json:"device_id"`
		Model              string            `json:"model"`
		SerialNumber       string            `json:"serial_number"`
		OverallStatus      enum.HealthStatus `json:"overall_status"`
		ReallocatedSectors int               `json:"reallocated_sectors"`
		ReadErrorRate      float64           `json:"read_error_rate"`
		Temperature        int               `json:"temperature"`
		PowerOnHours       int64             `json:"power_on_hours"`
		IOPSVariance       float64           `json:"iops_variance"`
		LatencyTrend       string            `json:"latency_trend"`
		ErrorTrend         string            `json:"error_trend"`
		PredictedFailure   bool              `json:"predicted_failure"`
		Timestamp          time.Time         `json:"timestamp"`
	}{
		DeviceID:           smartData.DeviceID,
		Model:              smartData.Model,
		SerialNumber:       smartData.SerialNumber,
		OverallStatus:      smartData.OverallStatus,
		ReallocatedSectors: smartData.ReallocatedSectors,
		ReadErrorRate:      smartData.ReadErrorRate,
		Temperature:        smartData.Temperature,
		PowerOnHours:       smartData.PowerOnHours,
		IOPSVariance:       metrics.IOPSVariance,
		LatencyTrend:       metrics.LatencyTrend,
		ErrorTrend:         metrics.ErrorTrend,
		PredictedFailure:   metrics.PredictedFailure,
		Timestamp:          metrics.Timestamp,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.Error("failed to encode prediction response", zap.Error(err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	logger.Info("served disk predictions",
		zap.String("device_id", deviceID),
		zap.String("status", smartData.OverallStatus.String()),
		zap.Bool("predicted_failure", metrics.PredictedFailure))
}

// handleDiskMetrics handles requests to /api/v1/disks/metrics for OpenMetrics output.
func (h *DiskHandler) handleDiskMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract device ID and time window from query parameters
	deviceID := r.URL.Query().Get("device_id")
	if deviceID == "" {
		http.Error(w, "device_id is required", http.StatusBadRequest)
		return
	}

	windowStr := r.URL.Query().Get("window")
	window, err := time.ParseDuration(windowStr)
	if err != nil || window <= 0 {
		window = 24 * time.Hour // Default to 24 hours
	}

	// Get prediction results
	smartData, metrics, err := h.predictor.PredictDiskFailure(deviceID, window)
	if err != nil {
		logger.Error("failed to get disk metrics",
			zap.String("device_id", deviceID),
			zap.Duration("window", window),
			zap.Error(err))
		if errors.Is(err, errors.NewNotFound("", nil)) {
			http.Error(w, "disk not found", http.StatusNotFound)
		} else {
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
		return
	}

	// Format metrics in OpenMetrics (Prometheus-compatible) text format
	var openMetrics strings.Builder
	openMetrics.WriteString("# HELP ioshelper_disk_reallocated_sectors Number of reallocated sectors on the disk\n")
	openMetrics.WriteString("# TYPE ioshelper_disk_reallocated_sectors gauge\n")
	openMetrics.WriteString(
		`ioshelper_disk_reallocated_sectors{device_id="` + deviceID + `",model="` + smartData.Model + `"} ` +
			strconv.Itoa(smartData.ReallocatedSectors) + "\n")

	openMetrics.WriteString("# HELP ioshelper_disk_read_error_rate Read error rate percentage\n")
	openMetrics.WriteString("# TYPE ioshelper_disk_read_error_rate gauge\n")
	openMetrics.WriteString(
		`ioshelper_disk_read_error_rate{device_id="` + deviceID + `",model="` + smartData.Model + `"} ` +
			strconv.FormatFloat(smartData.ReadErrorRate, 'f', 6, 64) + "\n")

	openMetrics.WriteString("# HELP ioshelper_disk_temperature_celsius Disk temperature in Celsius\n")
	openMetrics.WriteString("# TYPE ioshelper_disk_temperature_celsius gauge\n")
	openMetrics.WriteString(
		`ioshelper_disk_temperature_celsius{device_id="` + deviceID + `",model="` + smartData.Model + `"} ` +
			strconv.Itoa(smartData.Temperature) + "\n")

	openMetrics.WriteString("# HELP ioshelper_disk_power_on_hours Total power-on hours\n")
	openMetrics.WriteString("# TYPE ioshelper_disk_power_on_hours counter\n")
	openMetrics.WriteString(
		`ioshelper_disk_power_on_hours{device_id="` + deviceID + `",model="` + smartData.Model + `"} ` +
			strconv.FormatInt(smartData.PowerOnHours, 10) + "\n")

	openMetrics.WriteString("# HELP ioshelper_disk_iops_variance Variance in IOPS over time\n")
	openMetrics.WriteString("# TYPE ioshelper_disk_iops_variance gauge\n")
	openMetrics.WriteString(
		`ioshelper_disk_iops_variance{device_id="` + deviceID + `",model="` + smartData.Model + `"} ` +
			strconv.FormatFloat(metrics.IOPSVariance, 'f', 2, 64) + "\n")

	openMetrics.WriteString("# HELP ioshelper_disk_predicted_failure Predicted disk failure (1 for true, 0 for false)\n")
	openMetrics.WriteString("# TYPE ioshelper_disk_predicted_failure gauge\n")
	predictedFailure := 0
	if metrics.PredictedFailure {
		predictedFailure = 1
	}
	openMetrics.WriteString(
		`ioshelper_disk_predicted_failure{device_id="` + deviceID + `",model="` + smartData.Model + `"} ` +
			strconv.Itoa(predictedFailure) + "\n")

	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
