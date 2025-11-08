package usecase

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"time"

	// todo: not use gorm here
	"gorm.io/datatypes"

	error "github.com/webhook-processor/internal/shared/error"
	"github.com/webhook-processor/internal/webhook/domain"

	"github.com/webhook-processor/internal/shared/env"
	"github.com/webhook-processor/internal/shared/http"
	"github.com/webhook-processor/internal/shared/persistence/gorm"
)

func SendWebhook(msg domain.WebhookEventMessage) (bool, error.Error) {
	db := gorm.NewDB(gorm.DbOptions{
		Host:     env.GetEnvOrDefault("POSTGRES_HOST", "localhost"),
		DbName:   env.GetEnvOrDefault("POSTGRES_DB", "webhook_processor"),
		User:     env.GetEnvOrDefault("POSTGRES_USER", "webhook_user"),
		Password: env.GetEnvOrDefault("POSTGRES_PASSWORD", "webhook_pass"),
		Schema:   env.GetEnvOrDefault("POSTGRES_SCHEMA", "webhooks"),
	})

	ctx := context.Background()

	tx := db.WithContext(ctx).Begin()
	if tx.Error != nil {
		log.Fatalf("failed to begin transaction: %v\n", tx.Error)
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	event := domain.WebhookEvent{}
	result := tx.Model(&domain.WebhookEvent{}).Where("id = ?", msg.Id).First(&event)
	if result.Error != nil {
		if result.Error.Error() == "record not found" {
			log.Printf("no webhook found with id %s\n", event.Id)
			return false, domain.ErrWebhookEventDeliveryFailed(map[string]interface{}{
				"error": "webhook not found",
			})
		}

		log.Fatalf("query error: %v\n", result.Error)
		return false, domain.ErrWebhookEventDeliveryFailed(map[string]interface{}{
			"error": result.Error.Error(),
		})
	}

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

	// TODO: treat wb statu
	wb := domain.Webhook{}
	result = tx.Model(&domain.Webhook{}).Where("id = ?", event.WebhookId).First(&wb)
	if result.Error != nil {
		if result.Error.Error() == "record not found" {
			log.Printf("no webhook found with id %s\n", event.Id)
			return false, domain.ErrWebhookEventDeliveryFailed(map[string]interface{}{
				"error": "webhook not found",
			})
		}

		log.Fatalf("query error: %v\n", result.Error)
		return false, domain.ErrWebhookEventDeliveryFailed(map[string]interface{}{
			"error": result.Error.Error(),
		})
	}

	jsonBytes, err := json.Marshal(event.Payload)
	if err != nil {
		return false, domain.ErrWebhookEventPayloadSerializationFailed(map[string]interface{}{
			"error": err.Error(),
		})
	}
	reader := bytes.NewReader(jsonBytes)

	if res := tx.Model(&domain.WebhookEvent{}).Where("id = ?", event.Id).Update("retries_count", event.RetriesCount+1); res.Error != nil {
		log.Fatalf("update error: %v\n", res.Error)
		return false, domain.ErrWebhookEventDeliveryFailed(map[string]interface{}{
			"error": res.Error.Error(),
		})
	}

	event.RetriesCount++

	// TODO: treat timeout error
	httpClient := http.NewClient(http.ClientOpts{Timeout: time.Second * 5})
	res, err := httpClient.Post(wb.CallbackURL, "application/json", reader)
	if err != nil {
		return false, domain.ErrWebhookEventDeliveryFailed(map[string]interface{}{
			"error": err.Error(),
		})
	}
	defer res.Body.Close()

	resBodyBuffer, err := io.ReadAll(res.Body)
	if err != nil {
		return false, domain.ErrWebhookEventDeliveryFailed(map[string]interface{}{
			"error": err.Error(),
		})
	}

	var responseBody map[string]interface{}
	err = json.Unmarshal(resBodyBuffer, &responseBody)
	if err != nil {
		return false, domain.ErrWebhookEventDeliveryFailed(map[string]interface{}{
			"error": err.Error(),
		})
	}

	event.ResponseCode = res.StatusCode
	event.ResponseBody = datatypes.NewJSONType(responseBody)

	webhookWasSent := event.CheckSuccessResponse(res.StatusCode)
	if !webhookWasSent {
		event.MarkAsFailed(event.ResponseBody.Data())

		if res := tx.Model(&domain.WebhookEvent{}).Where("id = ?", event.Id).Updates(domain.WebhookEvent{
			Status:       event.Status,
			FailedAt:     event.FailedAt,
			LastError:    event.LastError,
			ResponseBody: event.ResponseBody,
			ResponseCode: event.ResponseCode,
		}); res.Error != nil {
			log.Fatalf("update error: %v\n", res.Error)
			return false, domain.ErrWebhookEventDeliveryFailed(map[string]interface{}{
				"error": res.Error.Error(),
			})
		}

		err := tx.Commit().Error
		if err != nil {
			log.Fatalf("commit error: %v\n", err)
			return false, domain.ErrWebhookEventDeliveryFailed(map[string]interface{}{
				"error": err.Error(),
			})
		}

		return false, error.New("Deu erro")
	}

	event.MarkAsDelivered()

	if res := tx.Model(&domain.WebhookEvent{}).Where("id = ?", event.Id).Updates(domain.WebhookEvent{
		Status:       event.Status,
		FailedAt:     event.FailedAt,
		LastError:    event.LastError,
		ResponseBody: event.ResponseBody,
		ResponseCode: event.ResponseCode,
	}); res.Error != nil {
		log.Fatalf("update error: %v\n", res.Error)
		return false, domain.ErrWebhookEventDeliveryFailed(map[string]interface{}{
			"error": res.Error.Error(),
		})
	}

	err = tx.Commit().Error
	if err != nil {
		log.Fatalf("commit error: %v\n", err)
		return false, domain.ErrWebhookEventDeliveryFailed(map[string]interface{}{
			"error": err.Error(),
		})
	}

	return true, error.New("")
}
