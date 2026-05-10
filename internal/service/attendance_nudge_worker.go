package service

import (
	"context"
	"encoding/json"
	"log"
	"meetBack/internal/model"
	"time"
)

type attendanceNudgeRepository interface {
	ListQueuedAttendanceNudges(ctx context.Context, limit int) ([]*model.AttendanceNudge, error)
	GetMeetingById(ctx context.Context, id uint32) (*model.Meeting, error)
	GetAttendanceReminderTargets(ctx context.Context, meetingId uint32) ([]*model.NotificationSubscription, error)
	MarkAttendanceNudgeSent(ctx context.Context, nudgeId string, sentAt time.Time) error
	MarkAttendanceNudgeFailed(ctx context.Context, nudgeId string) error
	MarkPushSubscriptionInvalid(ctx context.Context, subscriptionId uint32) error
	MarkPushSubscriptionSuppressed(ctx context.Context, subscriptionId uint32) error
}

type AttendanceNudgeWorker struct {
	repository   attendanceNudgeRepository
	pushSender   PushSender
	pollInterval time.Duration
	batchSize    int
	sendTimeout  time.Duration
}

func NewAttendanceNudgeWorker(repository attendanceNudgeRepository, pushSender PushSender) *AttendanceNudgeWorker {
	return &AttendanceNudgeWorker{
		repository:   repository,
		pushSender:   pushSender,
		pollInterval: 2 * time.Second,
		batchSize:    20,
		sendTimeout:  15 * time.Second,
	}
}

func (w *AttendanceNudgeWorker) Run(ctx context.Context) {
	if w == nil || w.repository == nil || w.pushSender == nil {
		return
	}

	w.processBatch(ctx)

	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.processBatch(ctx)
		}
	}
}

func (w *AttendanceNudgeWorker) processBatch(ctx context.Context) {
	nudges, err := w.repository.ListQueuedAttendanceNudges(ctx, w.batchSize)
	if err != nil {
		log.Printf("[attendance-nudge-worker] list queued nudges failed: %v", err)
		return
	}

	for _, nudge := range nudges {
		if nudge == nil {
			continue
		}

		if err := w.processNudge(ctx, nudge); err != nil {
			log.Printf("[attendance-nudge-worker] nudgeId=%s failed: %v", nudge.NudgeId, err)
		}
	}
}

func (w *AttendanceNudgeWorker) processNudge(ctx context.Context, nudge *model.AttendanceNudge) error {
	meeting, err := w.repository.GetMeetingById(ctx, nudge.MeetingId)
	if err != nil {
		_ = w.repository.MarkAttendanceNudgeFailed(ctx, nudge.NudgeId)
		return err
	}

	subscriptions, err := w.repository.GetAttendanceReminderTargets(ctx, nudge.MeetingId)
	if err != nil {
		_ = w.repository.MarkAttendanceNudgeFailed(ctx, nudge.NudgeId)
		return err
	}

	if len(subscriptions) == 0 {
		return w.repository.MarkAttendanceNudgeFailed(ctx, nudge.NudgeId)
	}

	payload, err := buildAttendanceNudgePayload(meeting, nudge)
	if err != nil {
		_ = w.repository.MarkAttendanceNudgeFailed(ctx, nudge.NudgeId)
		return err
	}

	successCount := 0
	for _, sub := range subscriptions {
		sendCtx, cancel := context.WithTimeout(ctx, w.sendTimeout)
		sendErr := w.pushSender.Send(sendCtx, sub, payload)
		cancel()
		if sendErr != nil {
			if stateErr := updatePushSubscriptionDeliveryState(ctx, w.repository, sub, sendErr); stateErr != nil {
				log.Printf(
					"[attendance-nudge-worker] nudgeId=%s deviceId=%s state update failed: %v",
					nudge.NudgeId,
					sub.DeviceId,
					stateErr,
				)
			}
			log.Printf(
				"[attendance-nudge-worker] nudgeId=%s deviceId=%s send failed: %v",
				nudge.NudgeId,
				sub.DeviceId,
				sendErr,
			)
			continue
		}
		successCount++
	}

	if successCount == 0 {
		return w.repository.MarkAttendanceNudgeFailed(ctx, nudge.NudgeId)
	}

	return w.repository.MarkAttendanceNudgeSent(ctx, nudge.NudgeId, time.Now().UTC())
}

func buildAttendanceNudgePayload(meeting *model.Meeting, nudge *model.AttendanceNudge) ([]byte, error) {
	body := nudge.MessageOverride
	if body == "" {
		body = "아직 가능한 시간을 선택하지 않았어요. 지금 미팅 응답을 남겨주세요."
	}

	return json.Marshal(map[string]any{
		"title": meeting.Title,
		"body":  body,
		"tag":   "meeting-" + meeting.ShortId + "-attendance-nudge",
		"data": map[string]any{
			"url":       "/m/" + meeting.ShortId,
			"meetingId": meeting.ID,
			"nudgeId":   nudge.NudgeId,
			"type":      "attendance-nudge",
		},
	})
}
