package enum

import "fmt"

// HealthStatus represents the health state of a component.
type HealthStatus int

const (
	Healthy HealthStatus = iota
	SubHealthy
	Failed
)

// String converts HealthStatus to a string representation.
func (s HealthStatus) String() string {
	switch s {
	case Healthy:
		return "healthy"
	case SubHealthy:
		return "subhealthy"
	case Failed:
		return "failed"
	default:
		return fmt.Sprintf("unknown(%d)", s)
	}
}

// DeviceType represents the type of monitored device.
type DeviceType int

const (
	RAID DeviceType = iota
	Disk
	Network
)

// String converts DeviceType to a string representation.
func (d DeviceType) String() string {
	switch d {
	case RAID:
		return "raid"
	case Disk:
		return "disk"
	case Network:
		return "network"
	default:
		return fmt.Sprintf("unknown(%d)", d)
	}
}

// IsolationStrategy represents the remediation isolation strategy.
type IsolationStrategy int

const (
	Temporary IsolationStrategy = iota
	Permanent
)

// String converts IsolationStrategy to a string representation.
func (i IsolationStrategy) String() string {
	switch i {
	case Temporary:
		return "temporary"
	case Permanent:
		return "permanent"
	default:
		return fmt.Sprintf("unknown(%d)", i)
	}
}