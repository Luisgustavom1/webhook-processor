package usecase

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net"
	"time"

	// todo: not use gorm here
	"gorm.io/datatypes"
	"gorm.io/gorm"

	error "github.com/webhook-processor/internal/shared/error"
	"github.com/webhook-processor/internal/shared/http"
	"github.com/webhook-processor/internal/webhook/domain"
)

func SendWebhook(db *gorm.DB, msg domain.WebhookEventMessage) (bool, error.Error) {
	ctx := context.Background()

	tx := db.WithContext(ctx).Begin()
	if tx.Error != nil {
		log.Fatalf("failed to begin transaction: %v\n", tx.Error)
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		} else {
			if err := tx.Commit().Error; err != nil {
				log.Fatalf("failed to commit transaction: %v\n", err)
			}
		}
	}()

	event := domain.WebhookEvent{}
	result := tx.Model(&domain.WebhookEvent{}).Where("id = ?", msg.Id).First(&event)
	if result.Error != nil {
		errMsg := result.Error.Error()
		if errMsg == "record not found" {
			log.Printf("no webhook found with id %s\n", event.Id)
			return false, domain.ErrWebhookEventDeliveryFailed(map[string]interface{}{
				"error": "webhook not found",
			})
		}

		log.Printf("query error: %v\n", errMsg)
		return false, domain.ErrWebhookEventDeliveryFailed(map[string]interface{}{
			"error": errMsg,
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

	var responseBody map[string]interface{}

	// TODO: treat timeout error
	httpClient := http.NewClient(http.ClientOpts{Timeout: time.Millisecond})
	res, err := httpClient.Post(wb.CallbackURL, "application/json", reader)
	if err != nil {
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			responseBody = map[string]interface{}{
				"error": "timeout",
				"cause": err.Error(),
			}
			event.ResponseCode = 408
		} else {
			return false, domain.ErrWebhookEventDeliveryFailed(map[string]interface{}{
				"error": err.Error(),
			})
		}
	} else if responseBody == nil {
		resBodyBuffer, err := io.ReadAll(res.Body)
		if err != nil {
			return false, domain.ErrWebhookEventDeliveryFailed(map[string]interface{}{
				"error": err.Error(),
			})
		}

		err = json.Unmarshal(resBodyBuffer, &responseBody)
		if err != nil {
			return false, domain.ErrWebhookEventDeliveryFailed(map[string]interface{}{
				"error": err.Error(),
			})
		}
		defer res.Body.Close()

		event.ResponseCode = res.StatusCode
		event.ResponseBody = datatypes.NewJSONType(responseBody)
	}

	webhookWasSent := event.CheckSuccessResponse(event.ResponseCode)
	if !webhookWasSent {
		event.MarkAsFailed(responseBody)

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

	return true, error.New("")
}
