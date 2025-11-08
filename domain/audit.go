package domain

import "time"

// AuditEntry represents an audit log entry for important domain actions
type AuditEntry struct {
    ID           string    `json:"id"`
    ResourceType string    `json:"resource_type"`
    ResourceID   string    `json:"resource_id"`
    Action       string    `json:"action"`
    ActorID      string    `json:"actor_id"`
    ActorRole    string    `json:"actor_role"`
    Metadata     map[string]string `json:"metadata,omitempty"`
    CreatedAt    time.Time `json:"created_at"`
}