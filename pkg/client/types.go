package client

import (
	"encoding/json"
	"time"
)

// Agent represents a Chronary agent resource.
type Agent struct {
	ID          string          `json:"id"`
	OrgID       string          `json:"orgId"`
	Name        string          `json:"name"`
	Type        string          `json:"type"`
	Description *string         `json:"description"`
	Status      string          `json:"status"`
	Metadata    json.RawMessage `json:"metadata"`
	CreatedAt   time.Time       `json:"createdAt"`
	UpdatedAt   time.Time       `json:"updatedAt"`
}

// ListResponse wraps a paginated list from the API.
type ListResponse[T any] struct {
	Data   []T `json:"data"`
	Total  int `json:"total"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

// Calendar represents a Chronary calendar resource.
//
// Casing mirrors the API response exactly: most fields are camelCase, but
// agent_status, ical_url, and default_reminders are snake_case because
// formatCalendar adds them on top of the raw row.
type Calendar struct {
	ID       string  `json:"id"`
	OrgID    string  `json:"orgId"`
	AgentID  *string `json:"agentId"`
	Name     string  `json:"name"`
	Timezone string  `json:"timezone"`
	// agent_status / ical_url are always present and non-empty in the REST
	// response, so omitempty is a no-op there. The nullable pointer fields
	// (agentId, externalId, provider, deletedAt) drop omitempty so a JSON
	// `null` mirrors the REST response shape exactly for `-o json`.
	AgentStatus      string          `json:"agent_status,omitempty"`
	ICalURL          string          `json:"ical_url,omitempty"`
	ExternalID       *string         `json:"externalId"`
	Provider         *string         `json:"provider"`
	DefaultReminders []int           `json:"default_reminders"`
	Metadata         json.RawMessage `json:"metadata"`
	DeletedAt        *string         `json:"deletedAt"`
	CreatedAt        time.Time       `json:"createdAt"`
	UpdatedAt        time.Time       `json:"updatedAt"`
}

// Event represents a Chronary event resource.
type Event struct {
	ID          string          `json:"id"`
	CalendarID  string          `json:"calendarId"`
	Title       string          `json:"title"`
	Description *string         `json:"description"`
	StartTime   time.Time       `json:"startTime"`
	EndTime     time.Time       `json:"endTime"`
	AllDay      bool            `json:"allDay"`
	Status      string          `json:"status"`
	Source      string          `json:"source"`
	Reminders   []int           `json:"reminders"`
	Metadata    json.RawMessage `json:"metadata"`
	// Nullable in REST (populated only for status="hold"); no omitempty so a
	// JSON `null` mirrors the REST shape.
	HoldExpiresAt *time.Time `json:"holdExpiresAt"`
	HoldPriority  *int       `json:"holdPriority"`
	// RecurrenceRule (RFC 5545 RRULE subset, no "RRULE:" prefix) is set on
	// recurring series masters; null for one-off events. RecurrenceExdates are
	// the ISO 8601 starts of individually cancelled occurrences (EXDATE).
	RecurrenceRule    *string  `json:"recurrenceRule"`
	RecurrenceExdates []string `json:"recurrenceExdates"`
	// Present only on expanded instances (expand=true): the series master id
	// and the occurrence start the instance was generated for.
	RecurringEventID  string    `json:"recurringEventId,omitempty"`
	OriginalStartTime string    `json:"originalStartTime,omitempty"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
}

// Webhook represents a Chronary webhook subscription.
type Webhook struct {
	ID                  string   `json:"id"`
	OrgID               string   `json:"orgId"`
	URL                 string   `json:"url"`
	Events              []string `json:"events"`
	Active              bool     `json:"active"`
	ConsecutiveFailures int      `json:"consecutiveFailures"`
	// firstFailureAt is nullable in REST (null until the circuit breaker trips);
	// no omitempty so a JSON `null` mirrors the REST shape. secret keeps
	// omitempty — it's only present in the create response, omitted on GET/LIST.
	FirstFailureAt *string   `json:"firstFailureAt"`
	Secret         string    `json:"secret,omitempty"`
	CreatedAt      time.Time `json:"createdAt"`
}

// WebhookDelivery represents a single delivery attempt record.
type WebhookDelivery struct {
	ID             string     `json:"id"`
	SubscriptionID string     `json:"subscription_id"`
	EventType      string     `json:"event_type"`
	Status         string     `json:"status"`
	Attempts       int        `json:"attempts"`
	LastAttemptAt  *time.Time `json:"last_attempt_at"`
	NextRetryAt    *time.Time `json:"next_retry_at"`
	CreatedAt      time.Time  `json:"created_at"`
	Payload        any        `json:"payload,omitempty"`
}

// WebhookDeliveryStats contains aggregate delivery counts.
type WebhookDeliveryStats struct {
	Pending   int `json:"pending"`
	Delivered int `json:"delivered"`
	Failed    int `json:"failed"`
}

// WebhookDeliveryListResponse is the response from GET /v1/webhooks/:id/deliveries.
type WebhookDeliveryListResponse struct {
	Data   []WebhookDelivery    `json:"data"`
	Total  int                  `json:"total"`
	Limit  int                  `json:"limit"`
	Offset int                  `json:"offset"`
	Stats  WebhookDeliveryStats `json:"stats"`
}

// AvailabilitySlot represents a free time slot.
type AvailabilitySlot struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

// BusyBlock represents a busy time block.
type BusyBlock struct {
	Start string  `json:"start"`
	End   string  `json:"end"`
	Title *string `json:"title,omitempty"`
}

// AvailabilityResponse is the shape returned by availability endpoints.
type AvailabilityResponse struct {
	Slots []AvailabilitySlot `json:"slots"`
	Busy  []BusyBlock        `json:"busy,omitempty"`
}

// UsageCounter tracks used vs limit for a resource.
type UsageCounter struct {
	Used  int `json:"used"`
	Limit int `json:"limit"`
}

// ScopedKeysUsage tracks active agent-scoped API keys for the current period.
// Distinct shape from UsageCounter: the count key is `count` (not `used`), and
// `limit` is nullable (null = unlimited on tiers without a key cap).
type ScopedKeysUsage struct {
	Count int  `json:"count"`
	Limit *int `json:"limit"`
}

// HoldsUsage tracks temporal-hold lifecycle counters for the current period.
// Informational — not gated by any plan limit. Funnel identity:
// `created = confirmed + expired + active` (active is derived).
type HoldsUsage struct {
	Created   int `json:"created"`
	Confirmed int `json:"confirmed"`
	Expired   int `json:"expired"`
}

// CrossCalendarQueriesUsage counts availability requests touching >1 calendar.
// Informational — gated separately by the cross_calendar_availability capability.
type CrossCalendarQueriesUsage struct {
	Used int `json:"used"`
}

// UsageResponse is the shape returned by GET /v1/usage.
type UsageResponse struct {
	PeriodStart          string                    `json:"period_start"`
	PeriodEnd            string                    `json:"period_end"`
	Plan                 string                    `json:"plan"`
	Agents               UsageCounter              `json:"agents"`
	Calendars            UsageCounter              `json:"calendars"`
	Events               UsageCounter              `json:"events"`
	APICalls             UsageCounter              `json:"api_calls"`
	Webhooks             UsageCounter              `json:"webhooks"`
	WebhookEndpoints     UsageCounter              `json:"webhook_endpoints"`
	AvailabilityQueries  UsageCounter              `json:"availability_queries"`
	ICalSubscriptions    UsageCounter              `json:"ical_subscriptions"`
	Proposals            UsageCounter              `json:"proposals"`
	RecurringEvents      UsageCounter              `json:"recurring_events"`
	ScopedKeys           ScopedKeysUsage           `json:"scoped_keys"`
	Holds                HoldsUsage                `json:"holds"`
	CrossCalendarQueries CrossCalendarQueriesUsage `json:"cross_calendar_queries"`
}

// ScopedAPIKey represents an agent-scoped API key.
type ScopedAPIKey struct {
	ID        string    `json:"id"`
	KeyPrefix string    `json:"key_prefix"`
	AgentID   string    `json:"agent_id"`
	Label     *string   `json:"label"`
	CreatedAt time.Time `json:"created_at"`
}

// CreatedScopedAPIKey is the create response for an agent-scoped API key.
type CreatedScopedAPIKey struct {
	ScopedAPIKey
	Key string `json:"key"`
}

// ScopedAPIKeyListResponse is the shape returned by GET /v1/keys.
type ScopedAPIKeyListResponse struct {
	Keys []ScopedAPIKey `json:"keys"`
}

// ICalSubscription represents an iCal feed subscription.
type ICalSubscription struct {
	ID           string    `json:"id"`
	AgentID      string    `json:"agentId"`
	CalendarID   string    `json:"calendarId"`
	URL          string    `json:"url"`
	Label        *string   `json:"label,omitempty"`
	Status       string    `json:"status"`
	LastSyncedAt *string   `json:"lastSyncedAt,omitempty"`
	LastError    *string   `json:"lastError,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
}

// HealthResponse is the shape returned by GET /health.
type HealthResponse struct {
	Status string `json:"status"`
	TS     string `json:"ts"`
}

// CalendarContext is the temporal snapshot for a calendar.
type CalendarContext struct {
	CalendarID   string  `json:"calendar_id"`
	Now          string  `json:"now"`
	AgentStatus  string  `json:"agent_status"`
	CurrentEvent *Event  `json:"current_event,omitempty"`
	NextEvent    *Event  `json:"next_event,omitempty"`
	RecentEvents []Event `json:"recent_events"`
	Upcoming     []Event `json:"upcoming"`
}

// AvailabilityRules is the buffer+working-hours config for a calendar.
type AvailabilityRules struct {
	ID                  string          `json:"id"`
	CalendarID          string          `json:"calendar_id"`
	BufferBeforeMinutes int             `json:"buffer_before_minutes"`
	BufferAfterMinutes  int             `json:"buffer_after_minutes"`
	WorkingHours        json.RawMessage `json:"working_hours,omitempty"`
	Timezone            string          `json:"timezone"`
	CreatedAt           time.Time       `json:"created_at"`
	UpdatedAt           time.Time       `json:"updated_at"`
}

// ProposalSlot is a candidate time slot for a scheduling proposal.
// The scheduling API emits snake_case keys (see formatSlot in the service).
type ProposalSlot struct {
	ID         *string   `json:"id,omitempty"`
	StartTime  time.Time `json:"start_time"`
	EndTime    time.Time `json:"end_time"`
	Weight     float64   `json:"weight"`
	CalendarID *string   `json:"calendar_id,omitempty"`
}

// ProposalResponse captures an agent's response to a proposal.
type ProposalResponse struct {
	ID             string         `json:"id"`
	AgentID        string         `json:"agent_id"`
	Response       string         `json:"response"`
	SelectedSlotID *string        `json:"selected_slot_id,omitempty"`
	CounterSlots   []ProposalSlot `json:"counter_slots,omitempty"`
	Message        *string        `json:"message,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
}

// Proposal represents a scheduling proposal with slots and responses.
type Proposal struct {
	ID                  string             `json:"id"`
	Title               string             `json:"title"`
	Description         *string            `json:"description,omitempty"`
	OrganizerAgentID    string             `json:"organizer_agent_id"`
	ParticipantAgentIDs []string           `json:"participant_agent_ids"`
	CalendarID          string             `json:"calendar_id"`
	Status              string             `json:"status"`
	ExpiresAt           *time.Time         `json:"expires_at,omitempty"`
	ResolvedSlot        *ProposalSlot      `json:"resolved_slot,omitempty"`
	CreatedEventID      *string            `json:"created_event_id,omitempty"`
	Metadata            json.RawMessage    `json:"metadata"`
	Slots               []ProposalSlot     `json:"slots,omitempty"`
	Responses           []ProposalResponse `json:"responses,omitempty"`
	CreatedAt           time.Time          `json:"created_at"`
	UpdatedAt           time.Time          `json:"updated_at"`
}

// ResolveProposalResult is the shape returned by POST .../resolve.
type ResolveProposalResult struct {
	Status       string        `json:"status"`
	ResolvedSlot *ProposalSlot `json:"resolved_slot,omitempty"`
	Reason       *string       `json:"reason,omitempty"`
}

// CancelProposalResult is the shape returned by POST .../cancel.
type CancelProposalResult struct {
	Status string `json:"status"`
}

// PlanLimits mirrors the machine-readable quota caps returned in /v1/plans.
// nil fields indicate unlimited.
type PlanLimits struct {
	Agents              *int `json:"agents"`
	Calendars           *int `json:"calendars"`
	Events              *int `json:"events"`
	APICalls            *int `json:"api_calls"`
	WebhookDeliveries   *int `json:"webhook_deliveries"`
	AvailabilityQueries *int `json:"availability_queries"`
	ICalSubscriptions   *int `json:"ical_subscriptions"`
	Proposals           *int `json:"proposals"`
	WebhookEndpoints    *int `json:"webhook_endpoints"`
	ScopedKeys          *int `json:"scoped_keys"`
}

// Plan is one tier in the public plan catalog.
type Plan struct {
	ID              string      `json:"id"`
	Name            string      `json:"name"`
	Tagline         string      `json:"tagline"`
	Price           *int        `json:"price"`
	Currency        *string     `json:"currency"`
	Limits          *PlanLimits `json:"limits"`
	DisplayFeatures []string    `json:"display_features"`
	Recommended     bool        `json:"recommended"`
	CustomPricing   bool        `json:"custom_pricing,omitempty"`
	ContactURL      string      `json:"contact_url,omitempty"`
}

// PlansListResponse is the shape returned by GET /v1/plans.
type PlansListResponse struct {
	Plans []Plan `json:"plans"`
}

// AuditLogEntry is a single audit-log record.
type AuditLogEntry struct {
	ID             string  `json:"id"`
	Action         string  `json:"action"`
	ActorKeyPrefix *string `json:"actor_key_prefix"`
	AgentID        *string `json:"agent_id"`
	Resource       *string `json:"resource"`
	IP             *string `json:"ip"`
	Status         int     `json:"status"`
	Method         string  `json:"method"`
	Path           string  `json:"path"`
	DurationMS     int     `json:"duration_ms"`
	RequestID      *string `json:"request_id"`
	CreatedAt      string  `json:"created_at"`
}

// AuditLogResponse is the shape returned by GET /v1/audit-log.
type AuditLogResponse struct {
	Data       []AuditLogEntry `json:"data"`
	Pagination struct {
		NextCursor *string `json:"next_cursor"`
	} `json:"pagination"`
	RetentionDays *int `json:"retention_days"`
	RangeClamped  bool `json:"range_clamped"`
}
