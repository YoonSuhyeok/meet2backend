package service

import (
	"context"
	"errors"
	"meetBack/internal/model"
	"strconv"
)

type pushEndpointStateRepository interface {
	MarkPushSubscriptionInvalid(ctx context.Context, subscriptionId uint32) error
	MarkPushSubscriptionSuppressed(ctx context.Context, subscriptionId uint32) error
}

type pushDeliveryAction string

const (
	pushDeliveryActionNone       pushDeliveryAction = ""
	pushDeliveryActionInvalid    pushDeliveryAction = "invalid"
	pushDeliveryActionSuppressed pushDeliveryAction = "sending_suppressed"
)

type PushDeliveryError struct {
	StatusCode   int
	ResponseBody string
	Err          error
}

func (e *PushDeliveryError) Error() string {
	if e == nil {
		return "push delivery error"
	}
	if e.StatusCode > 0 {
		if e.ResponseBody != "" {
			return "web push failed with status " + strconv.Itoa(e.StatusCode) + ": " + e.ResponseBody
		}
		return "web push failed with status " + strconv.Itoa(e.StatusCode)
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return "push delivery error"
}

func (e *PushDeliveryError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func classifyPushDeliveryError(err error) pushDeliveryAction {
	var deliveryErr *PushDeliveryError
	if !errors.As(err, &deliveryErr) || deliveryErr == nil {
		return pushDeliveryActionNone
	}

	switch deliveryErr.StatusCode {
	case 404, 410:
		return pushDeliveryActionInvalid
	case 400, 405, 406, 409, 412, 413, 422:
		return pushDeliveryActionSuppressed
	default:
		return pushDeliveryActionNone
	}
}

func updatePushSubscriptionDeliveryState(
	ctx context.Context,
	repository pushEndpointStateRepository,
	subscription *model.NotificationSubscription,
	err error,
) error {
	if repository == nil || subscription == nil || err == nil {
		return nil
	}

	switch classifyPushDeliveryError(err) {
	case pushDeliveryActionInvalid:
		return repository.MarkPushSubscriptionInvalid(ctx, subscription.ID)
	case pushDeliveryActionSuppressed:
		return repository.MarkPushSubscriptionSuppressed(ctx, subscription.ID)
	default:
		return nil
	}
}
