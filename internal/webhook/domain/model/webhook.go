package model

import (
	"time"

	"github.com/lib/pq"
)

const WEBHOOK_QUEUE = "webhook_queue"
const EXCHANGE_NAME = "webhook_exchange"
const ROUTING_KEY = "webhook.process"
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

func (w *Webhook) IsActive() bool {
	return w.Status == WebhookStatusActive
}
