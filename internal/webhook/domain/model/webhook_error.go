package model

import (
	error "github.com/webhook-processor/internal/shared/error"
)

var (
	// webhook
	ErrWebhookNotFound = func(args ...interface{}) error.Error {
		return error.New("webhook not found", args...)
	}
	ErrWebhookIsDisabled = func(args ...interface{}) error.Error {
		return error.New("webhook is disabled", args...)
	}

	// webhook event
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
	ErrWebhookEventNotFound = func(args ...interface{}) error.Error {
		return error.New("webhook event not found", args...)
	}
)
