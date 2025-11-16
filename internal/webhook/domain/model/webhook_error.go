package model

import "fmt"

type WebhookError struct {
	error
	Retryable bool
}

func (e *WebhookError) IsRetryable() bool {
	return e.Retryable
}

func New(err error, retryable bool) *WebhookError {
	return &WebhookError{
		error:     err,
		Retryable: retryable,
	}
}

func newError(message string, args ...interface{}) error {
	return fmt.Errorf(message, args)
}

var (
	// webhook
	ErrWebhookNotFound = func(args ...interface{}) *WebhookError {
		return New(newError("webhook not found", args...), false)
	}
	ErrWebhookIsDisabled = func(args ...interface{}) *WebhookError {
		return New(newError("webhook is disabled", args...), false)
	}

	// webhook event
	ErrWebhookEventNotPending = func(args ...interface{}) *WebhookError {
		return New(newError("webhook event is not pending", args...), false)
	}
	ErrWebhookEventReachedMaxAttempts = func(args ...interface{}) *WebhookError {
		return New(newError("webhook event reached max attempts", args...), false)
	}
	ErrWebhookEventPayloadSerializationFailed = func(args ...interface{}) *WebhookError {
		return New(newError("webhook event payload serialization failed", args...), false)
	}
	ErrWebhookEventDeliveryFailed = func(args ...interface{}) *WebhookError {
		return New(newError("webhook event delivery failed", args...), false)
	}
	ErrWebhookEventNotFound = func(args ...interface{}) *WebhookError {
		return New(newError("webhook event not found", args...), false)
	}
	ErrWebhookEventFails = func(args ...interface{}) *WebhookError {
		return New(newError("webhook event fails and marked as failed", args...), false)
	}
)
