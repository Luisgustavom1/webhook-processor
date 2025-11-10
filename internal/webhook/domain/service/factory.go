package service

import (
	"github.com/webhook-processor/internal/webhook/ports"
)

type webhookService struct {
	repo ports.WebhookRepositoryPort
}

func NewWebhookService(repo ports.WebhookRepositoryPort) *webhookService {
	return &webhookService{
		repo: repo,
	}
}
