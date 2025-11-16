package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"time"

	log "github.com/webhook-processor/internal/shared/logger"

	"github.com/webhook-processor/internal/shared/http"
	"github.com/webhook-processor/internal/webhook/domain/model"
)

func (s *webhookService) SendWebhook(msg model.WebhookEventMessage) (event *model.WebhookEvent, errWb *model.WebhookError) {
	ctx := context.Background()
	trx := s.repo.Transaction(&ctx)
	defer trx.Rollback(&ctx)

	event, err := s.repo.GetWebhookEventByID(ctx, msg.Id)
	if err != nil {
		log.Error("query error", "err", err.Error())
		return event, model.ErrWebhookEventDeliveryFailed(map[string]interface{}{
			"error": err.Error(),
		})
	}
	if event == nil {
		log.Info("no webhook found with id", "id", msg.Id)
		return event, model.ErrWebhookEventNotFound(map[string]interface{}{
			"id": msg.Id,
		})
	}

	wb, err := s.repo.GetWebhookByID(ctx, event.WebhookId)
	if err != nil {
		log.Error("query error", "err", err.Error())
		return event, model.ErrWebhookEventDeliveryFailed(map[string]interface{}{
			"error": err.Error(),
		})
	}
	if wb == nil {
		log.Info("no webhook found with id", "id", event.WebhookId)
		return event, model.ErrWebhookEventNotFound(map[string]interface{}{
			"id": event.WebhookId,
		})
	}

	if !event.IsPending() {
		log.Info("webhook not is pending", "status", event.Status)
		return event, model.ErrWebhookEventNotPending("status", event.Status)
	}

	if event.ReachedMaxAttempts() {
		return event, model.ErrWebhookEventReachedMaxAttempts(
			map[string]interface{}{
				"tries": event.Tries,
			},
		)
	}

	if !wb.IsActive() {
		return event, model.ErrWebhookIsDisabled(map[string]interface{}{
			"error": "webhook is not active",
		})
	}

	event.Tries++
	if err := s.repo.UpdateWebhookEventById(ctx, event.Id, model.WebhookEvent{
		Tries: event.Tries,
	}); err != nil {
		log.Error("update error", "err", err)
		return event, model.ErrWebhookEventDeliveryFailed(map[string]interface{}{
			"error": err.Error(),
		})
	}

	err = trx.Commit(&ctx)
	if err != nil {
		log.Error("commit error", "err", err.Error())
		return event, model.ErrWebhookEventDeliveryFailed(map[string]interface{}{
			"error": err.Error(),
		})
	}

	jsonBytes, err := json.Marshal(event.Payload)
	if err != nil {
		// send to dead letter
		if err := s.repo.UpdateWebhookEventById(ctx, event.Id, model.WebhookEvent{
			Status: model.WebhookEventsStatusDeadLetter,
		}); err != nil {
			log.Error("update error", "err", err)
			return event, model.ErrWebhookEventDeliveryFailed(map[string]interface{}{
				"error": err.Error(),
			})
		}

		return event, model.ErrWebhookEventPayloadSerializationFailed(map[string]interface{}{
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

	if responseBody == nil && netErr == nil {
		resBodyBuffer, err := io.ReadAll(res.Body)
		defer res.Body.Close()
		if err != nil {
			return event, model.ErrWebhookEventDeliveryFailed(map[string]interface{}{
				"error": err.Error(),
			})
		}

		err = json.Unmarshal(resBodyBuffer, &responseBody)
		if err != nil {
			return event, model.ErrWebhookEventDeliveryFailed(map[string]interface{}{
				"error": err.Error(),
			})
		}

		event.ResponseCode = res.StatusCode
		event.SetResponseBody(responseBody)
	}

	sentSuccessfully := event.CheckSuccessResponse(event.ResponseCode) && netErr == nil
	if sentSuccessfully {
		event.MarkAsDelivered()

		if err := s.repo.UpdateWebhookEventById(ctx, event.Id, *event); err != nil {
			log.Error("update error", "err", err)
			return event, model.ErrWebhookEventDeliveryFailed(map[string]interface{}{
				"error": err.Error(),
			})
		}

		return event, nil
	}

	event.MarkAsFailed(responseBody)

	if err := s.repo.UpdateWebhookEventById(ctx, event.Id, *event); err != nil {
		log.Error("update error", "err", err)
		return event, model.ErrWebhookEventDeliveryFailed(map[string]interface{}{
			"error": err.Error(),
		})
	}

	return event, model.ErrWebhookEventFails()
}
