package service

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"

	log "github.com/webhook-processor/internal/shared/logger"

	"github.com/webhook-processor/internal/shared/http"
	"github.com/webhook-processor/internal/webhook/domain/model"
)

func (s *webhookService) SendWebhook(ctx context.Context, msg model.WebhookEventMessage) (event *model.WebhookEvent, errWb *model.WebhookError) {
	event, wb, errWb := s.getAndValidatePreconditions(ctx, msg)
	if errWb != nil {
		return event, errWb
	}

	trx := s.repo.Transaction(&ctx)
	defer trx.Rollback(&ctx)

	event.Tries++
	if err := s.repo.UpdateWebhookEventById(ctx, event.Id, model.WebhookEvent{
		Tries: event.Tries,
	}); err != nil {
		log.Error("update error on tries increment", "err", err)
		return event, model.ErrWebhookEventDeliveryFailed(map[string]interface{}{"error": err.Error()})
	}

	if err := trx.Commit(&ctx); err != nil {
		log.Error("commit error on tries increment", "err", err.Error())
		return event, model.ErrWebhookEventDeliveryFailed(map[string]interface{}{"error": err.Error()})
	}

	jsonBytes, err := json.Marshal(event.Payload)
	if err != nil {
		return s.markAsDeadLetter(ctx, event, err)
	}

	reader := bytes.NewReader(jsonBytes)
	signature, err := s.generateHMACSignature(event.Payload.Data(), wb.Secret)
	if err != nil {
		return s.markAsDeadLetter(ctx, event, err)
	}

	res, err := s.httpClient.Post(wb.CallbackURL, "application/json", reader, map[string]string{
		"x-signature": signature,
	})

	responseBody, responseCode, netErr := s.parseHttpResponse(res, err)
	event.ResponseCode = responseCode
	if responseBody != nil {
		event.SetResponseBody(responseBody)
	}

	sentSuccessfully := event.CheckSuccessResponse(event.ResponseCode) && netErr == nil
	if sentSuccessfully {
		event.MarkAsDelivered()
	} else if !event.IsRetryableCode() || event.ReachedMaxAttempts() {
		event.MarkAsFailed(responseBody)
	}

	if err := s.repo.UpdateWebhookEventById(ctx, event.Id, *event); err != nil {
		log.Error("update error on final state", "err", err)
		return event, model.ErrWebhookEventDeliveryFailed(map[string]interface{}{"error": err.Error()})
	}

	if sentSuccessfully {
		return event, nil
	}

	if event.Status == model.WebhookEventsStatusFailed {
		return event, model.ErrWebhookEventFails()
	}

	return event, model.ErrWebhookEventWillRetry(event.ResponseCode)
}

func (s *webhookService) getAndValidatePreconditions(ctx context.Context, msg model.WebhookEventMessage) (*model.WebhookEvent, *model.Webhook, *model.WebhookError) {
	event, err := s.repo.GetWebhookEventByID(ctx, msg.Id)
	if err != nil {
		log.Error("query error", "err", err.Error())
		return event, nil, model.ErrWebhookEventDeliveryFailed(map[string]interface{}{"error": err.Error()})
	}
	if event == nil {
		log.Info("no webhook found with id", "id", msg.Id)
		return event, nil, model.ErrWebhookEventNotFound(map[string]interface{}{"id": msg.Id})
	}

	wb, err := s.repo.GetWebhookByID(ctx, event.WebhookId)
	if err != nil {
		log.Error("query error", "err", err.Error())
		return event, nil, model.ErrWebhookEventDeliveryFailed(map[string]interface{}{"error": err.Error()})
	}
	if wb == nil {
		log.Info("no webhook found with id", "id", event.WebhookId)
		return event, nil, model.ErrWebhookEventNotFound(map[string]interface{}{"id": event.WebhookId})
	}

	if !event.IsPending() {
		log.Info("webhook not is pending", "status", event.Status)
		return event, wb, model.ErrWebhookEventNotPending("status", event.Status)
	}
	if event.ReachedMaxAttempts() {
		return event, wb, model.ErrWebhookEventReachedMaxAttempts(map[string]interface{}{"tries": event.Tries})
	}
	if !wb.IsActive() {
		return event, wb, model.ErrWebhookIsDisabled(map[string]interface{}{"error": "webhook is not active"})
	}

	return event, wb, nil
}

func (s *webhookService) parseHttpResponse(res *http.Response, err error) (body map[string]interface{}, statusCode int, netErr net.Error) {
	var timeoutErr bool
	if err != nil {
		timeoutErr = errors.As(err, &netErr) && netErr.Timeout()
	}

	if timeoutErr {
		body = map[string]interface{}{
			"error": "timeout",
			"cause": err.Error(),
		}
		statusCode = 408
		return
	}

	if err != nil {
		body = map[string]interface{}{
			"error": "network error",
			"cause": err.Error(),
		}
		statusCode = 503
		return
	}

	defer res.Body.Close()
	statusCode = res.StatusCode

	resBodyBuffer, err := io.ReadAll(res.Body)
	if err != nil {
		body = map[string]interface{}{"error": "failed to read response body"}
		return
	}

	if err := json.Unmarshal(resBodyBuffer, &body); err != nil {
		body = map[string]interface{}{"raw_response": string(resBodyBuffer)}
	}

	return
}

func (s *webhookService) markAsDeadLetter(ctx context.Context, event *model.WebhookEvent, serializationError error) (*model.WebhookEvent, *model.WebhookError) {
	if err := s.repo.UpdateWebhookEventById(ctx, event.Id, model.WebhookEvent{
		Status: model.WebhookEventsStatusDeadLetter,
	}); err != nil {
		log.Error("update error to dead letter failed", "err", err)
		return event, model.ErrWebhookEventDeliveryFailed(map[string]interface{}{"error": err.Error()})
	}

	return event, model.ErrWebhookEventPayloadSerializationFailed(map[string]interface{}{
		"error": serializationError.Error(),
	})
}

func (s *webhookService) generateHMACSignature(payload model.Object, secret string) (string, error) {
	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	sig := hmac.New(sha256.New, []byte(secret))
	sig.Write(jsonBytes)

	return fmt.Sprintf("sha256=%x", sig.Sum(nil)), nil
}
