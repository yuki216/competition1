package domain

import "time"

// MetricFilter defines filters used to compute metrics over tickets
type MetricFilter struct {
    Start       time.Time       `json:"start,omitempty"`
    End         time.Time       `json:"end,omitempty"`
    Status      []TicketStatus  `json:"status,omitempty"`
    Category    []TicketCategory `json:"category,omitempty"`
    AssignedTo  string          `json:"assigned_to,omitempty"`
}

// Metric represents calculated metrics for reporting/analytics
type Metric struct {
    ResolutionTimes     []time.Duration   `json:"resolution_times,omitempty"`
    FirstResponseTimes  []time.Duration   `json:"first_response_times,omitempty"`
    TicketCounts        map[string]int    `json:"ticket_counts,omitempty"`
    GeneratedAt         time.Time         `json:"generated_at"`
}