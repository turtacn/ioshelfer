// api/v1/network.go
package v1

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"internal/common/logger"
	"internal/core/detection"
)

// NetworkAPI handles network-related REST API endpoints.
type NetworkAPI struct {
	logger        *logger.Logger
	detector      *detection.NetworkDetector
	webhookURL    string
	webhookClient *http.Client
	mu            sync.Mutex
}

// NetworkHealthResponse represents the response for the /api/v1/network/health endpoint.
type NetworkHealthResponse struct {
	Timestamp   string  `json:"timestamp"`
	LatencyMs   float64 `json:"latency_ms"`
	PacketLoss  float64 `json:"packet_loss_percent"`
	IsHealthy   bool    `json:"is_healthy"`
	Error       string  `json:"error,omitempty"`
}

// WebhookPayload represents the payload sent to the webhook.
type WebhookPayload struct {
	Event     string              `json:"event"`
	Timestamp string              `json:"timestamp"`
	Health    NetworkHealthResponse `json:"health"`
}

// NewNetworkAPI creates a new NetworkAPI instance.
func NewNetworkAPI(logger *logger.Logger, detector *detection.NetworkDetector, webhookURL string) *NetworkAPI {
	return &NetworkAPI{
		logger:        logger,
		detector:      detector,
		webhookURL:    webhookURL,
		webhookClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// RegisterRoutes registers the API routes with the provided HTTP router.
func (n *NetworkAPI) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/network/health", n.handleHealth)
}

// handleHealth handles the /api/v1/network/health endpoint.
func (n *NetworkAPI) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		n.logger.Error("Method not allowed for /api/v1/network/health", "method", r.Method)
		return
	}

	// Perform network health check
	latency, packetLoss, err := n.detector.CheckHealth()
	response := NetworkHealthResponse{
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
		LatencyMs:  latency,
		PacketLoss: packetLoss,
		IsHealthy:  err == nil,
	}

	if err != nil {
		response.Error = err.Error()
		n.logger.Error("Network health check failed", "error", err)
	} else {
		n.logger.Info("Network health check completed", "latency_ms", latency, "packet_loss_percent", packetLoss)
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		w.WriteHeader(http.StatusOK)
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		n.logger.Error("Failed to encode response", "error", err)
		return
	}

	// Send webhook notification if configured
	if n.webhookURL != "" {
		n.sendWebhookNotification(response)
	}
}

// sendWebhookNotification sends a webhook notification with the health check results.
func (n *NetworkAPI) sendWebhookNotification(health NetworkHealthResponse) {
	n.mu.Lock()
	defer n.mu.Unlock()

	payload := WebhookPayload{
		Event:     "network_health_check",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Health:    health,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		n.logger.Error("Failed to marshal webhook payload", "error", err)
		return
	}

	req, err := http.NewRequest(http.MethodPost, n.webhookURL, bytes.NewBuffer(body))
	if err != nil {
		n.logger.Error("Failed to create webhook request", "error", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := n.webhookClient.Do(req)
	if err != nil {
		n.logger.Error("Failed to send webhook notification", "error", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		n.logger.Error("Webhook notification failed", "status_code", resp.StatusCode)
	} else {
		n.logger.Info("Webhook notification sent successfully")
	}
}
