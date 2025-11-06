package domain

import (
	"time"
)

// Metric represents system performance metrics
type Metric struct {
	TotalTickets           int           `json:"total_tickets"`
	OpenTickets            int           `json:"open_tickets"`
	ResolvedTickets        int           `json:"resolved_tickets"`
	AverageResolutionTime  time.Duration `json:"average_resolution_time"`
	FirstResponseTime      time.Duration `json:"first_response_time"`
	AIAccuracyRate         float64       `json:"ai_accuracy_rate"`
	SLAComplianceRate      float64       `json:"sla_compliance_rate"`
	TotalKnowledgeEntries  int           `json:"total_knowledge_entries"`
	ActiveKnowledgeEntries int           `json:"active_knowledge_entries"`
	Period                 string        `json:"period"`
	GeneratedAt            time.Time     `json:"generated_at"`
}

// MetricPeriod represents the period for metrics calculation
type MetricPeriod string

const (
	MetricPeriodDaily   MetricPeriod = "daily"
	MetricPeriodWeekly  MetricPeriod = "weekly"
	MetricPeriodMonthly MetricPeriod = "monthly"
)

// MetricFilter represents filters for metrics
type MetricFilter struct {
	Period    MetricPeriod `json:"period"`
	StartDate *time.Time   `json:"start_date,omitempty"`
	EndDate   *time.Time   `json:"end_date,omitempty"`
	Category  *string      `json:"category,omitempty"`
}

// NewMetric creates a new metric instance
func NewMetric(period string) *Metric {
	return &Metric{
		Period:      period,
		GeneratedAt: time.Now(),
	}
}

// CalculateResolutionTime calculates the average resolution time
func (m *Metric) CalculateResolutionTime(resolutionTimes []time.Duration) {
	if len(resolutionTimes) == 0 {
		return
	}

	var total time.Duration
	for _, rt := range resolutionTimes {
		total += rt
	}
	m.AverageResolutionTime = total / time.Duration(len(resolutionTimes))
}

// CalculateAIAccuracy calculates AI accuracy rate based on feedback
func (m *Metric) CalculateAIAccuracy(totalSuggestions, successfulSuggestions int) {
	if totalSuggestions == 0 {
		return
	}
	m.AIAccuracyRate = float64(successfulSuggestions) / float64(totalSuggestions)
}

// CalculateSLACompliance calculates SLA compliance rate
func (m *Metric) CalculateSLACompliance(totalTickets, compliantTickets int) {
	if totalTickets == 0 {
		return
	}
	m.SLAComplianceRate = float64(compliantTickets) / float64(totalTickets)
}

// AuditEntry represents an audit log entry
type AuditEntry struct {
	ID         string                 `json:"id"`
	ResourceID string                 `json:"resource_id"`
	ResourceType string               `json:"resource_type"`
	Action     string                 `json:"action"`
	ActorID    string                 `json:"actor_id"`
	ActorRole  string                 `json:"actor_role"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt  time.Time              `json:"created_at"`
}

// NewAuditEntry creates a new audit entry
func NewAuditEntry(resourceID, resourceType, action, actorID, actorRole string) *AuditEntry {
	return &AuditEntry{
		ID:           generateAuditID(),
		ResourceID:   resourceID,
		ResourceType: resourceType,
		Action:       action,
		ActorID:      actorID,
		ActorRole:    actorRole,
		Metadata:     make(map[string]interface{}),
		CreatedAt:    time.Now(),
	}
}

// AddMetadata adds metadata to the audit entry
func (a *AuditEntry) AddMetadata(key string, value interface{}) {
	if a.Metadata == nil {
		a.Metadata = make(map[string]interface{})
	}
	a.Metadata[key] = value
}

// Metric errors
var (
	ErrInvalidMetricPeriod = NewDomainError("invalid metric period")
	ErrInvalidDateRange    = NewDomainError("invalid date range")
)

// Helper function for generating audit IDs
func generateAuditID() string {
	return "audit_" + time.Now().Format("20060102150405")
}