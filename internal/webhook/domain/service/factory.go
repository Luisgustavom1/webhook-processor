package service

import (
	"github.com/webhook-processor/internal/shared/http"
	"github.com/webhook-processor/internal/webhook/ports"
)

type webhookService struct {
	repo       ports.WebhookRepositoryPort
	httpClient *http.HTTPClient
}

func NewWebhookService(repo ports.WebhookRepositoryPort, httpClient *http.HTTPClient) *webhookService {
	return &webhookService{
		repo:       repo,
		httpClient: httpClient,
	}
}
