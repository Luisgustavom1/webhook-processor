package domain

import (
	"time"

	"github.com/lib/pq"
	"gorm.io/datatypes"
)

const WEBHOOK_QUEUE = "email_queue"
const MAX_WEBHOOK_SEND_ATTEMPTS = 5

type WebhookStatus string

const (
	WebhookStatusActive   WebhookStatus = "active"
	WebhookStatusDisabled WebhookStatus = "disabled"
)

type Webhook struct {
	Id               int            `json:"id"`
	FailureCount     int            `json:"failure_count"`
	CallbackURL      string         `json:"callback_url"`
	Secret           string         `json:"secret"`
	Status           WebhookStatus  `json:"status"`
	LastFailureAt    time.Time      `json:"last_failure_at"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	SubscribedEvents pq.StringArray `json:"subscribed_events" gorm:"type:text[]"`
}

type WebhookEventsStatus string

const (
	WebhookEventsStatusPending    WebhookEventsStatus = "pending"
	WebhookEventsStatusDelivered  WebhookEventsStatus = "delivered"
	WebhookEventsStatusFailed     WebhookEventsStatus = "failed"
	WebhookEventsStatusDeadLetter WebhookEventsStatus = "dead_letter"
)

type object = map[string]interface{}

type WebhookEvent struct {
	Id           string                     `json:"id"`
	WebhookId    int                        `json:"webhook_id"`
	EventType    string                     `json:"event_type"`
	Payload      datatypes.JSONType[object] `json:"payload"`
	LastError    datatypes.JSONType[object] `json:"last_error"`
	ResponseBody datatypes.JSONType[object] `json:"response_body"`
	ResponseCode int                        `json:"response_code"`
	RetriesCount int                        `json:"retries_count"`
	Status       WebhookEventsStatus        `json:"status"`
	FailedAt     time.Time                  `json:"failed_at,omitempty"`
	DeliveredAt  time.Time                  `json:"delivered_at,omitempty"`
	CreatedAt    time.Time                  `json:"created_at"`
	UpdatedAt    time.Time                  `json:"updated_at"`
}

type WebhookEventMessage struct {
	Id string `json:"id"`
}

func NewWebhookEvent(wb WebhookEvent) *WebhookEvent {
	return &WebhookEvent{
		Id:           wb.Id,
		EventType:    wb.EventType,
		Payload:      wb.Payload,
		LastError:    wb.LastError,
		ResponseBody: wb.ResponseBody,
		ResponseCode: wb.ResponseCode,
		RetriesCount: wb.RetriesCount,
		Status:       wb.Status,
		FailedAt:     wb.FailedAt,
		DeliveredAt:  wb.DeliveredAt,
		CreatedAt:    wb.CreatedAt,
		UpdatedAt:    wb.UpdatedAt,
	}
}

func (wb *WebhookEvent) IsPending() bool {
	return wb.Status == WebhookEventsStatusPending
}

func (wb *WebhookEvent) ReachedMaxAttempts() bool {
	return wb.RetriesCount >= MAX_WEBHOOK_SEND_ATTEMPTS
}

func (wb *WebhookEvent) CheckSuccessResponse(code int) bool {
	// any 2xx is a success
	return code/100 == 2
}

func (wb *WebhookEvent) MarkAsDelivered() {
	wb.Status = WebhookEventsStatusDelivered
	wb.DeliveredAt = time.Now()
}

func (wb *WebhookEvent) MarkAsFailed(error map[string]interface{}) {
	wb.LastError = datatypes.NewJSONType(error)
	wb.Status = WebhookEventsStatusFailed
	wb.FailedAt = time.Now()
}
