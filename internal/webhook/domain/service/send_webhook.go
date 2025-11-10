package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"time"

	error "github.com/webhook-processor/internal/shared/error"
	log "github.com/webhook-processor/internal/shared/logger"

	"github.com/webhook-processor/internal/shared/http"
	"github.com/webhook-processor/internal/webhook/domain/model"
)

func (s *webhookService) SendWebhook(msg model.WebhookEventMessage) (bool, error.Error) {
	ctx := context.Background()

	event, err := s.repo.GetWebhookEventByID(ctx, msg.Id)
	if err != nil {
		log.Error("query error", "err", err.Error())
		return false, model.ErrWebhookEventDeliveryFailed(map[string]interface{}{
			"error": err.Error(),
		})
	}

	if event == nil {
		log.Info("no webhook found with id", "id", msg.Id)
		return false, model.ErrWebhookEventNotFound(map[string]interface{}{
			"id": msg.Id,
		})
	}

	if !event.IsPending() {
		return false, model.ErrWebhookEventNotPending(map[string]interface{}{
			"status": event.Status,
		})
	}

	if event.ReachedMaxAttempts() {
		return false, model.ErrWebhookEventReachedMaxAttempts(
			map[string]interface{}{
				"tries": event.Tries,
			},
		)
	}

	wb, err := s.repo.GetWebhookByID(ctx, event.WebhookId)
	if err != nil {
		log.Error("query error", "err", err.Error())
		return false, model.ErrWebhookEventDeliveryFailed(map[string]interface{}{
			"error": err.Error(),
		})
	}

	if wb == nil {
		log.Info("no webhook found with id", "id", event.WebhookId)
		return false, model.ErrWebhookEventNotFound(map[string]interface{}{
			"id": event.WebhookId,
		})
	}

	if !wb.IsActive() {
		return false, model.ErrWebhookIsDisabled(map[string]interface{}{
			"error": "webhook is not active",
		})
	}

	event.Tries++
	if err := s.repo.UpdateWebhookEventById(ctx, event.Id, model.WebhookEvent{
		Tries: event.Tries,
	}); err != nil {
		log.Error("update error", "err", err.Error())
		return false, model.ErrWebhookEventDeliveryFailed(map[string]interface{}{
			"error": err.Error(),
		})
	}

	jsonBytes, err := json.Marshal(event.Payload)
	if err != nil {
		return false, model.ErrWebhookEventPayloadSerializationFailed(map[string]interface{}{
			"error": err.Error(),
		})
	}
	reader := bytes.NewReader(jsonBytes)

	var responseBody map[string]interface{}

	httpClient := http.NewClient(http.ClientOpts{Timeout: time.Second * 5})
	res, err := httpClient.Post(wb.CallbackURL, "application/json", reader)

	var netErr net.Error
	timeoutErr := err != nil && errors.As(err, &netErr) && netErr.Timeout()
	if timeoutErr {
		responseBody = map[string]interface{}{
			"error": "timeout",
			"cause": err.Error(),
		}
		event.ResponseCode = 408
	}

	if err != nil && !timeoutErr {
		return false, model.ErrWebhookEventDeliveryFailed(map[string]interface{}{
			"error": err.Error(),
		})
	}

	if responseBody == nil {
		resBodyBuffer, err := io.ReadAll(res.Body)
		if err != nil {
			return false, model.ErrWebhookEventDeliveryFailed(map[string]interface{}{
				"error": err.Error(),
			})
		}

		err = json.Unmarshal(resBodyBuffer, &responseBody)
		if err != nil {
			return false, model.ErrWebhookEventDeliveryFailed(map[string]interface{}{
				"error": err.Error(),
			})
		}
		defer res.Body.Close()

		event.ResponseCode = res.StatusCode
		event.SetResponseBody(responseBody)
	}

	sentSuccessfully := event.CheckSuccessResponse(event.ResponseCode)
	if sentSuccessfully {
		event.MarkAsDelivered()

		if err := s.repo.UpdateWebhookEventById(ctx, event.Id, model.WebhookEvent{
			Status:       event.Status,
			ResponseBody: event.ResponseBody,
			ResponseCode: event.ResponseCode,
		}); err != nil {
			log.Error("update error", "err", err)
			return false, model.ErrWebhookEventDeliveryFailed(map[string]interface{}{
				"error": err.Error(),
			})
		}

		return true, error.New("")
	}

	event.MarkAsFailed(responseBody)

	if err := s.repo.UpdateWebhookEventById(ctx, event.Id, model.WebhookEvent{
		Status:       event.Status,
		FailedAt:     event.FailedAt,
		LastError:    event.LastError,
		ResponseCode: event.ResponseCode,
	}); err != nil {
		log.Error("update error", "err", err)
		return false, model.ErrWebhookEventDeliveryFailed(map[string]interface{}{
			"error": err.Error(),
		})
	}

	return false, error.New("Deu erro")
}
