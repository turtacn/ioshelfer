// api/v1/raid.go
package v1

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"github.com/turtacn/ioshelfer/internal/common/errors"
	"github.com/turtacn/ioshelfer/internal/common/logger"
	"github.com/turtacn/ioshelfer/internal/common/types/enum"
	"github.com/turtacn/ioshelfer/internal/core/detection"
	"github.com/turtacn/ioshelfer/pkg/raid"
	"go.uber.org/zap"
)

// RAIDHandler handles RAID-related API requests.
type RAIDHandler struct {
	detector *detection.Detector
}

// NewRAIDHandler creates a new RAIDHandler instance.
func NewRAIDHandler(detector *detection.Detector) *RAIDHandler {
	return &RAIDHandler{
		detector: detector,
	}
}

// RegisterRoutes registers RAID-related API routes.
func (h *RAIDHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/raid/controllers/health", h.handleControllersHealth)
	mux.HandleFunc("/api/v1/raid/controllers/metrics", h.handleControllersMetrics)
}

// handleControllersHealth handles requests to /api/v1/raid/controllers/health.
func (h *RAIDHandler) handleControllersHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract controller ID from query parameters
	controllerID := r.URL.Query().Get("controller_id")
	if controllerID == "" {
		http.Error(w, "controller_id is required", http.StatusBadRequest)
		return
	}

	// Perform health check using the detector
	healthStatus, err := h.detector.CheckRAIDHealth(controllerID)
	if err != nil {
		logger.Error("failed to check RAID controller health",
			zap.String("controller_id", controllerID),
			zap.Error(err))
		if errors.Is(err, errors.NewNotFound("", nil)) {
			http.Error(w, "controller not found", http.StatusNotFound)
		} else {
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
		return
	}

	// Prepare response
	response := struct {
		ControllerID   string            `json:"controller_id"`
		Status         enum.HealthStatus `json:"status"`
		QueueDepth     int               `json:"queue_depth"`
		AvgLatency     string            `json:"avg_latency"`
		FirmwareStatus string            `json:"firmware_status"`
		Confidence     float64           `json:"confidence"`
		Recommendation string            `json:"recommendation"`
	}{
		ControllerID:   healthStatus.ControllerID,
		Status:         healthStatus.Status,
		QueueDepth:     healthStatus.QueueDepth,
		AvgLatency:     healthStatus.AvgLatency.String(),
		FirmwareStatus: healthStatus.FirmwareStatus,
		Confidence:     healthStatus.Confidence,
		Recommendation: healthStatus.Recommendation,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.Error("failed to encode health response", zap.Error(err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	logger.Info("served RAID controller health",
		zap.String("controller_id", controllerID),
		zap.String("status", healthStatus.Status.String()))
}

// handleControllersMetrics handles requests to /api/v1/raid/controllers/metrics for Prometheus metrics.
func (h *RAIDHandler) handleControllersMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract controller ID from query parameters
	controllerID := r.URL.Query().Get("controller_id")
	if controllerID == "" {
		http.Error(w, "controller_id is required", http.StatusBadRequest)
		return
	}

	// Get RAID metrics
	metrics, err := h.detector.GetRAIDMetrics(controllerID)
	if err != nil {
		logger.Error("failed to get RAID controller metrics",
			zap.String("controller_id", controllerID),
			zap.Error(err))
		if errors.Is(err, errors.NewNotFound("", nil)) {
			http.Error(w, "controller not found", http.StatusNotFound)
		} else {
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
		return
	}

	// Format metrics in Prometheus text format
	var prometheusMetrics strings.Builder
	prometheusMetrics.WriteString("# HELP ioshelper_raid_queue_depth Current queue depth of the RAID controller\n")
	prometheusMetrics.WriteString("# TYPE ioshelper_raid_queue_depth gauge\n")
	prometheusMetrics.WriteString(
		`ioshelper_raid_queue_depth{controller_id="` + controllerID + `"} ` +
			strconv.Itoa(metrics.QueueDepth) + "\n")

	prometheusMetrics.WriteString("# HELP ioshelper_raid_avg_latency_ms Average I/O latency in milliseconds\n")
	prometheusMetrics.WriteString("# TYPE ioshelper_raid_avg_latency_ms gauge\n")
	prometheusMetrics.WriteString(
		`ioshelper_raid_avg_latency_ms{controller_id="` + controllerID + `"} ` +
			strconv.FormatFloat(float64(metrics.AvgLatency.Milliseconds()), 'f', 2, 64) + "\n")

	prometheusMetrics.WriteString("# HELP ioshelper_raid_error_retry_rate Error retry rate per hour\n")
	prometheusMetrics.WriteString("# TYPE ioshelper_raid_error_retry_rate gauge\n")
	prometheusMetrics.WriteString(
		`ioshelper_raid_error_retry_rate{controller_id="` + controllerID + `"} ` +
			strconv.Itoa(metrics.ErrorRetryRate) + "\n")

	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	w.Write([]byte(prometheusMetrics.String()))

	logger.Info("served RAID controller metrics",
		zap.String("controller_id", controllerID))
}
