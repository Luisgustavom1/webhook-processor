package ports

import (
	"context"

	"github.com/webhook-processor/internal/webhook/domain/model"
)

type WebhookServicePort interface {
	SendWebhook(ctx context.Context, msg model.WebhookEventMessage) (*model.WebhookEvent, *model.WebhookError)
}
