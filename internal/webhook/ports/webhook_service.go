package ports

import (
	"context"

	repo "github.com/webhook-processor/internal/webhook/adapters/repo"
	"github.com/webhook-processor/internal/webhook/domain/model"
)

type WebhookServicePort interface {
	SendWebhook(ctx context.Context, msg model.WebhookEventMessage) (*model.WebhookEvent, *model.WebhookError)
}

type WebhookRepositoryPort interface {
	GetWebhookByID(ctx context.Context, id int) (*model.Webhook, error)
	GetWebhookEventByID(ctx context.Context, id string) (*model.WebhookEvent, error)
	UpdateWebhookEventById(ctx context.Context, id string, event model.WebhookEvent) error
	Transaction(ctx *context.Context) repo.MyTransaction
}
