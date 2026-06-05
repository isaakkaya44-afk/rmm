package shared

// RMM Platform Shared Constants
const (
	Version = "1.0.0"
)

// Ticket status constants
const (
	TicketStatusOpen          = "open"
	TicketStatusInProgress    = "in_progress"
	TicketStatusWaitingCustomer = "waiting_customer"
	TicketStatusResolved      = "resolved"
	TicketStatusClosed        = "closed"
)

// Ticket priority constants
const (
	PriorityLow    = "low"
	PriorityMedium = "medium"
	PriorityHigh   = "high"
	PriorityCritical = "critical"
)

// Alert severity constants
const (
	SeverityInfo     = "info"
	SeverityWarning  = "warning"
	SeverityCritical = "critical"
)

// Alert status constants
const (
	AlertStatusOpen         = "open"
	AlertStatusAcknowledged = "acknowledged"
	AlertStatusResolved     = "resolved"
)

// Device status
const (
	DeviceStatusOnline  = "online"
	DeviceStatusOffline = "offline"
)
