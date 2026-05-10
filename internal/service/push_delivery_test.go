package service

import (
	"context"
	"testing"

	"meetBack/internal/model"
)

type pushEndpointStateRepositoryStub struct {
	invalidSubscriptionIds    []uint32
	suppressedSubscriptionIds []uint32
}

func (r *pushEndpointStateRepositoryStub) MarkPushSubscriptionInvalid(_ context.Context, subscriptionId uint32) error {
	r.invalidSubscriptionIds = append(r.invalidSubscriptionIds, subscriptionId)
	return nil
}

func (r *pushEndpointStateRepositoryStub) MarkPushSubscriptionSuppressed(_ context.Context, subscriptionId uint32) error {
	r.suppressedSubscriptionIds = append(r.suppressedSubscriptionIds, subscriptionId)
	return nil
}

func TestClassifyPushDeliveryError(t *testing.T) {
	testCases := []struct {
		name     string
		err      error
		expected pushDeliveryAction
	}{
		{
			name: "expired endpoint becomes invalid",
			err: &PushDeliveryError{
				StatusCode: 410,
			},
			expected: pushDeliveryActionInvalid,
		},
		{
			name: "endpoint validation failure becomes suppressed",
			err: &PushDeliveryError{
				StatusCode: 413,
			},
			expected: pushDeliveryActionSuppressed,
		},
		{
			name: "auth failure is not endpoint specific",
			err: &PushDeliveryError{
				StatusCode: 401,
			},
			expected: pushDeliveryActionNone,
		},
		{
			name: "network error does not change endpoint state",
			err: &PushDeliveryError{
				Err: context.DeadlineExceeded,
			},
			expected: pushDeliveryActionNone,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if actual := classifyPushDeliveryError(tc.err); actual != tc.expected {
				t.Fatalf("expected %q, got %q", tc.expected, actual)
			}
		})
	}
}

func TestUpdatePushSubscriptionDeliveryState(t *testing.T) {
	repository := &pushEndpointStateRepositoryStub{}
	subscription := &model.NotificationSubscription{ID: 42}

	if err := updatePushSubscriptionDeliveryState(
		context.Background(),
		repository,
		subscription,
		&PushDeliveryError{StatusCode: 404},
	); err != nil {
		t.Fatalf("updatePushSubscriptionDeliveryState returned error: %v", err)
	}

	if len(repository.invalidSubscriptionIds) != 1 || repository.invalidSubscriptionIds[0] != 42 {
		t.Fatalf("expected subscription 42 to be invalidated, got %v", repository.invalidSubscriptionIds)
	}

	repository = &pushEndpointStateRepositoryStub{}
	if err := updatePushSubscriptionDeliveryState(
		context.Background(),
		repository,
		subscription,
		&PushDeliveryError{StatusCode: 422},
	); err != nil {
		t.Fatalf("updatePushSubscriptionDeliveryState returned error: %v", err)
	}

	if len(repository.suppressedSubscriptionIds) != 1 || repository.suppressedSubscriptionIds[0] != 42 {
		t.Fatalf("expected subscription 42 to be suppressed, got %v", repository.suppressedSubscriptionIds)
	}

	repository = &pushEndpointStateRepositoryStub{}
	if err := updatePushSubscriptionDeliveryState(
		context.Background(),
		repository,
		subscription,
		&PushDeliveryError{StatusCode: 503},
	); err != nil {
		t.Fatalf("updatePushSubscriptionDeliveryState returned error: %v", err)
	}

	if len(repository.invalidSubscriptionIds) != 0 || len(repository.suppressedSubscriptionIds) != 0 {
		t.Fatalf("expected no state change for retryable failure, got invalid=%v suppressed=%v", repository.invalidSubscriptionIds, repository.suppressedSubscriptionIds)
	}
}
