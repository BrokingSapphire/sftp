// Package audit holds request/response DTOs for audit logs and telemetry.
package audit

// TelemetryRequest is a single UI interaction event from the frontend.
type TelemetryRequest struct {
	EventType string         `json:"event_type" validate:"required"` // click, view, navigate, search ...
	Element   string         `json:"element"`
	Path      string         `json:"path"`
	Metadata  map[string]any `json:"metadata"`
}

// LogResponse is the public projection of an audit log entry.
type LogResponse struct {
	ID         int64          `json:"id"`
	ActorID    string         `json:"actor_id,omitempty"`
	ActorEmail string         `json:"actor_email,omitempty"`
	Action     string         `json:"action"`
	Category   string         `json:"category"`
	ObjectType string         `json:"object_type,omitempty"`
	ObjectID   string         `json:"object_id,omitempty"`
	ObjectName string         `json:"object_name,omitempty"`
	Result     string         `json:"result"`
	IPAddress  string         `json:"ip_address,omitempty"`
	Browser    string         `json:"browser,omitempty"`
	OS         string         `json:"os,omitempty"`
	CreatedAt  string         `json:"created_at"`
}

// ListMeta is pagination metadata.
type ListMeta struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}
