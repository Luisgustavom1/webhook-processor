package model

import (
	"strconv"
	"time"

	"gorm.io/datatypes"
)

type WebhookEventsStatus string

const (
	WebhookEventsStatusPending    WebhookEventsStatus = "pending"
	WebhookEventsStatusDelivered  WebhookEventsStatus = "delivered"
	WebhookEventsStatusFailed     WebhookEventsStatus = "failed"
	WebhookEventsStatusDeadLetter WebhookEventsStatus = "dead_letter"
)

type object = map[string]interface{}

type WebhookEventMessage struct {
	Id string `json:"id"`
}

type WebhookEvent struct {
	Id           string                     `json:"id"`
	WebhookId    int                        `json:"webhook_id"`
	EventType    string                     `json:"event_type"`
	Payload      datatypes.JSONType[object] `json:"payload"`
	LastError    datatypes.JSONType[object] `json:"last_error"`
	ResponseBody datatypes.JSONType[object] `json:"response_body"`
	ResponseCode int                        `json:"response_code"`
	Tries        int                        `json:"tries"`
	Status       WebhookEventsStatus        `json:"status"`
	FailedAt     time.Time                  `json:"failed_at,omitempty"`
	DeliveredAt  time.Time                  `json:"delivered_at,omitempty"`
	CreatedAt    time.Time                  `json:"created_at"`
	UpdatedAt    time.Time                  `json:"updated_at"`
}

func (wb *WebhookEvent) IsPending() bool {
	return wb.Status == WebhookEventsStatusPending
}

func (wb *WebhookEvent) ReachedMaxAttempts() bool {
	return wb.Tries >= MAX_WEBHOOK_SEND_ATTEMPTS
}

func (wb *WebhookEvent) CheckSuccessResponse(code int) bool {
	// any 2xx is a success
	return code/100 == 2
}

var RETRYABLE_STATUS_CODE = map[string]bool{
	"408": true,
	"429": true,
	"502": true,
	"503": true,
	"504": true,
}

func (wb *WebhookEvent) IsRetryableCode() bool {
	return RETRYABLE_STATUS_CODE[strconv.Itoa(wb.ResponseCode)]
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

func (wb *WebhookEvent) SetResponseBody(responseBody object) {
	wb.ResponseBody = datatypes.NewJSONType(responseBody)
}
