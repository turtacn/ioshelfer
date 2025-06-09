package errors

import (
	"fmt"
	"github.com/pkg/errors"
)

// Error codes for specific failure scenarios.
const (
	ErrCodeQueueOverflow     = "ERR_QUEUE_OVERFLOW"
	ErrCodeFirmwareMismatch  = "ERR_FIRMWARE_MISMATCH"
	ErrCodeHighLatency       = "ERR_HIGH_LATENCY"
	ErrCodeStorageFailure    = "ERR_STORAGE_FAILURE"
	ErrCodeNetworkPacketLoss = "ERR_NETWORK_PACKET_LOSS"
)

// CustomError wraps an error with a specific code and message.
type CustomError struct {
	Code    string
	Message string
	Cause   error
}

// Error implements the error interface.
func (e *CustomError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the underlying cause of the error.
func (e *CustomError) Unwrap() error {
	return e.Cause
}

// NewQueueOverflow creates a new error for queue overflow scenarios.
func NewQueueOverflow(msg string, cause error) error {
	return &CustomError{
		Code:    ErrCodeQueueOverflow,
		Message: msg,
		Cause:   cause,
	}
}

// NewFirmwareMismatch creates a new error for firmware mismatch scenarios.
func NewFirmwareMismatch(msg string, cause error) error {
	return &CustomError{
		Code:    ErrCodeFirmwareMismatch,
		Message: msg,
		Cause:   cause,
	}
}

// NewHighLatency creates a new error for high latency scenarios.
func NewHighLatency(msg string, cause error) error {
	return &CustomError{
		Code:    ErrCodeHighLatency,
		Message: msg,
		Cause:   cause,
	}
}

// NewStorageFailure creates a new error for storage failure scenarios.
func NewStorageFailure(msg string, cause error) error {
	return &CustomError{
		Code:    ErrCodeStorageFailure,
		Message: msg,
		Cause:   cause,
	}
}

// NewNetworkPacketLoss creates a new error for network packet loss scenarios.
func NewNetworkPacketLoss(msg string, cause error) error {
	return &CustomError{
		Code:    ErrCodeNetworkPacketLoss,
		Message: msg,
		Cause:   cause,
	}
}

// Is checks if the target error matches the CustomError by code.
func Is(err, target error) bool {
	if customErr, ok := err.(*CustomError); ok {
		if targetCustom, ok := target.(*CustomError); ok {
			return customErr.Code == targetCustom.Code
		}
	}
	return errors.Is(err, target)
}

// Wrap adds context to an existing error while preserving its code.
func Wrap(err error, msg string) error {
	if err == nil {
		return nil
	}
	if customErr, ok := err.(*CustomError); ok {
		return &CustomError{
			Code:    customErr.Code,
			Message: fmt.Sprintf("%s: %s", msg, customErr.Message),
			Cause:   customErr.Cause,
		}
	}
	return errors.Wrap(err, msg)
}

func New(msg string, err error) error {
	return Wrap(err, msg)
}
