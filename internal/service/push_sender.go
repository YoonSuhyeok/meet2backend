package service

import (
	"context"
	"fmt"
	"io"
	"meetBack/internal/model"
	"strings"

	webpush "github.com/SherClockHolmes/webpush-go"
)

type WebPushSender struct {
	subscriber      string
	vapidPublicKey  string
	vapidPrivateKey string
	ttlSeconds      int
}

func NewWebPushSender(subscriber, vapidPublicKey, vapidPrivateKey string) *WebPushSender {
	return &WebPushSender{
		subscriber:      strings.TrimSpace(subscriber),
		vapidPublicKey:  strings.TrimSpace(vapidPublicKey),
		vapidPrivateKey: strings.TrimSpace(vapidPrivateKey),
		ttlSeconds:      60,
	}
}

func (s *WebPushSender) Send(
	ctx context.Context,
	subscription *model.NotificationSubscription,
	payload []byte,
) error {
	if subscription == nil {
		return fmt.Errorf("subscription is nil")
	}

	resp, err := webpush.SendNotification(payload, &webpush.Subscription{
		Endpoint: subscription.Endpoint,
		Keys: webpush.Keys{
			Auth:   subscription.Auth,
			P256dh: subscription.P256dh,
		},
	}, &webpush.Options{
		Subscriber:      s.subscriber,
		VAPIDPublicKey:  s.vapidPublicKey,
		VAPIDPrivateKey: s.vapidPrivateKey,
		TTL:             s.ttlSeconds,
	})
	if err != nil {
		return &PushDeliveryError{Err: err}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return &PushDeliveryError{
			StatusCode:   resp.StatusCode,
			ResponseBody: strings.TrimSpace(string(body)),
		}
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}
