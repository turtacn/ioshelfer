package storage

import (
	"encoding/json"
	"github.com/turtacn/ioshelfer/internal/common/errors"
	"github.com/turtacn/ioshelfer/internal/common/logger"
	"github.com/turtacn/ioshelfer/internal/common/types/enum"
	"go.uber.org/zap"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Metric represents a single data point for storage.
type Metric struct {
	Timestamp  time.Time       `json:"timestamp"`
	DeviceType enum.DeviceType `json:"device_type"`
	DeviceID   string          `json:"device_id"`
	Value      interface{}     `json:"value"`
}

// Storage defines the interface for storing and querying metrics.
type Storage interface {
	Store(metric Metric) error
	Query(deviceType enum.DeviceType, deviceID string, window time.Duration) ([]Metric, error)
}

// FileStorage implements Storage using a file-based backend.
type FileStorage struct {
	baseDir string
	mu      sync.RWMutex
}

// NewFileStorage creates a new FileStorage instance.
func NewFileStorage(baseDir string) (*FileStorage, error) {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, errors.NewStorageFailure("failed to create storage directory", err)
	}
	return &FileStorage{
		baseDir: baseDir,
	}, nil
}

// Store saves a metric to the file-based storage.
func (s *FileStorage) Store(metric Metric) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	filePath := s.getFilePath(metric.DeviceType, metric.DeviceID)
	var metrics []Metric

	// Read existing metrics if file exists
	if data, err := os.ReadFile(filePath); err == nil {
		if err := json.Unmarshal(data, &metrics); err != nil {
			return errors.NewStorageFailure("failed to unmarshal existing metrics", err)
		}
	} else if !os.IsNotExist(err) {
		return errors.NewStorageFailure("failed to read storage file", err)
	}

	// Append new metric
	metrics = append(metrics, metric)

	// Write back to file
	data, err := json.MarshalIndent(metrics, "", "  ")
	if err != nil {
		return errors.NewStorageFailure("failed to marshal metrics", err)
	}
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return errors.NewStorageFailure("failed to write metrics to file", err)
	}

	logger.Info("stored metric",
		zap.String("device_type", metric.DeviceType.String()),
		zap.String("device_id", metric.DeviceID),
		zap.Time("timestamp", metric.Timestamp),
	)
	return nil
}

// Query retrieves metrics for a device within a time window.
func (s *FileStorage) Query(deviceType enum.DeviceType, deviceID string, window time.Duration) ([]Metric, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	filePath := s.getFilePath(deviceType, deviceID)
	var metrics []Metric

	// Read metrics from file
	if data, err := os.ReadFile(filePath); err == nil {
		if err := json.Unmarshal(data, &metrics); err != nil {
			return nil, errors.NewStorageFailure("failed to unmarshal metrics", err)
		}
	} else if os.IsNotExist(err) {
		return []Metric{}, nil // No data is not an error
	} else {
		return nil, errors.NewStorageFailure("failed to read storage file", err)
	}

	// Filter metrics within the time window
	cutoff := time.Now().Add(-window)
	var result []Metric
	for _, m := range metrics {
		if m.Timestamp.After(cutoff) {
			result = append(result, m)
		}
	}

	logger.Info("queried metrics",
		zap.String("device_type", deviceType.String()),
		zap.String("device_id", deviceID),
		zap.Int("count", len(result)),
		zap.Duration("window", window),
	)
	return result, nil
}

// getFilePath generates the file path for a device's metrics.
func (s *FileStorage) getFilePath(deviceType enum.DeviceType, deviceID string) string {
	return filepath.Join(s.baseDir, deviceType.String(), deviceID+".json")
}
