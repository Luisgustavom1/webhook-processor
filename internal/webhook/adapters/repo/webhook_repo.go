package repo

import (
	"context"

	"github.com/webhook-processor/internal/webhook/domain/model"
	"gorm.io/gorm"
)

type WebhookRepo struct {
	db *gorm.DB
}

type MyTransaction struct {
	db *gorm.DB
}

type Transaction interface {
	Commit() error
	Rollback() error
}

func NewWebhookRepo(db *gorm.DB) *WebhookRepo {
	return &WebhookRepo{db: db}
}

func (r *WebhookRepo) GetWebhookByID(ctx context.Context, id int) (*model.Webhook, error) {
	var webhook model.Webhook
	if err := r.getDb(ctx).First(&webhook, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &webhook, nil
}

func (r *WebhookRepo) GetWebhookEventByID(ctx context.Context, id string) (*model.WebhookEvent, error) {
	var event model.WebhookEvent
	if err := r.getDb(ctx).First(&event, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &event, nil
}

func (r *WebhookRepo) UpdateWebhookEventById(ctx context.Context, id string, event model.WebhookEvent) error {
	return r.getDb(ctx).Model(&model.WebhookEvent{}).Where("id = ?", id).Updates(event).Error
}

func (r *WebhookRepo) Transaction(ctx context.Context) Transaction {
	trx := r.db.Begin()

	return MyTransaction{db: trx}
}

func (trx MyTransaction) Commit() error {
	return trx.db.Commit().Error
}

func (trx MyTransaction) Rollback() error {
	return trx.db.Rollback().Error
}

func (r *WebhookRepo) getDb(ctx context.Context) *gorm.DB {
	tx := ctx.Value("trx")

	if tx == nil {
		return r.db.WithContext(ctx)
	}

	return tx.(MyTransaction).db
}
