package repo

import (
	"context"

	"github.com/webhook-processor/internal/webhook/domain/model"
	"gorm.io/gorm"
)

type WebhookRepo struct {
	db *gorm.DB
}

func NewWebhookRepo(db *gorm.DB) *WebhookRepo {
	return &WebhookRepo{db: db}
}

func (r *WebhookRepo) GetWebhookByID(ctx context.Context, id int) (*model.Webhook, error) {
	var webhook model.Webhook
	if err := r.db.WithContext(ctx).First(&webhook, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &webhook, nil
}

func (r *WebhookRepo) GetWebhookEventByID(ctx context.Context, id string) (*model.WebhookEvent, error) {
	var event model.WebhookEvent
	if err := r.db.WithContext(ctx).First(&event, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &event, nil
}

func (r *WebhookRepo) UpdateWebhookEventById(ctx context.Context, id string, event model.WebhookEvent) error {
	return r.db.WithContext(ctx).Model(&model.WebhookEvent{}).Where("id = ?", id).Updates(event).Error
}
