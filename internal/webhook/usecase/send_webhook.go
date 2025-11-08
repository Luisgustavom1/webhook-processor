package usecase

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"time"

	error "github.com/webhook-processor/internal/shared/error"
	"github.com/webhook-processor/internal/webhook/domain"

	"github.com/webhook-processor/internal/shared/env"
	"github.com/webhook-processor/internal/shared/http"
	"github.com/webhook-processor/internal/shared/persistence/postgres"
)

func SendWebhook(event domain.WebhookEvent) (bool, error.Error) {
	if !event.IsPending() {
		return false, domain.ErrWebhookEventNotPending(map[string]interface{}{
			"status": event.Status,
		})
	}

	if event.ReachedMaxAttempts() {
		return false, domain.ErrWebhookEventReachedMaxAttempts(
			map[string]interface{}{
				"tries": event.RetriesCount,
			},
		)
	}

	db := postgres.NewPostgres(postgres.DbOptions{
		Host:     env.GetEnvOrDefault("POSTGRES_HOST", "localhost"),
		DbName:   env.GetEnvOrDefault("POSTGRES_DB", "webhook_processor"),
		User:     env.GetEnvOrDefault("POSTGRES_USER", "webhook_user"),
		Password: env.GetEnvOrDefault("POSTGRES_PASSWORD", "webhook_pass"),
	})

	wb := domain.Webhook{}
	ctx := context.Background()

	tx, err := db.BeginTx(ctx, &sql.TxOptions{})
	err = tx.QueryRowContext(ctx, "SELECT * FROM webhook WHERE id=?", event.Id).Scan(&wb)
	switch {
	case err == sql.ErrNoRows:
		log.Printf("now webhook found with id %d\n", event.Id)
	case err != nil:
		log.Fatalf("query error: %v\n", err)
	}

	jsonBytes, err := json.Marshal(event.Payload)
	if err != nil {
		return false, domain.ErrWebhookEventPayloadSerializationFailed(map[string]interface{}{
			"error": err.Error(),
		})
	}
	reader := bytes.NewReader(jsonBytes)

	event.RetriesCount++
	err = tx.QueryRowContext(ctx, "UPDATE webhook SET retries_count = retries_count + 1 WHERE id=?", event.Id).Scan(&wb)
	if err != nil {
		log.Fatalf("query error: %v\n", err)
	}

	// TODO: treat timeout error
	httpClient := http.NewClient(http.ClientOpts{Timeout: time.Second * 5})
	res, err := httpClient.Post(wb.CallbackURL, "application/json", reader)
	if err != nil {
		return false, domain.ErrWebhookEventDeliveryFailed(map[string]interface{}{
			"error": err.Error(),
		})
	}
	defer res.Body.Close()

	// save response data
	resBodyBuffer, err := io.ReadAll(res.Body)
	if err != nil {
		return false, domain.ErrWebhookEventDeliveryFailed(map[string]interface{}{
			"error": err.Error(),
		})
	}
	err = json.Unmarshal(resBodyBuffer, &event.ResponseBody)
	if err != nil {
		return false, domain.ErrWebhookEventDeliveryFailed(map[string]interface{}{
			"error": err.Error(),
		})
	}
	event.ResponseCode = res.StatusCode

	webhookWasSent := event.CheckSuccessResponse()
	if !webhookWasSent {
		event.MarkAsFailed(event.ResponseBody)

		return false, error.New("Deu erro")
	}

	event.MarkAsDelivered()

	return true, error.New("")
}
