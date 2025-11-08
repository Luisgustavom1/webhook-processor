package domain

import (
	error "github.com/webhook-processor/internal/shared/error"
)

var (
	ErrWebhookEventNotPending = func(args ...interface{}) error.Error {
		return error.New("webhook event is not pending", args...)
	}
	ErrWebhookEventReachedMaxAttempts = func(args ...interface{}) error.Error {
		return error.New("webhook event reached max attempts", args...)
	}
	ErrWebhookEventPayloadSerializationFailed = func(args ...interface{}) error.Error {
		return error.New("webhook event payload serialization failed", args...)
	}
	ErrWebhookEventDeliveryFailed = func(args ...interface{}) error.Error {
		return error.New("webhook event delivery failed", args...)
	}
)
