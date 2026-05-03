package repository

import (
	"context"
	"database/sql"
	"fmt"
	"meetBack/internal/model"

	"github.com/uptrace/bun"
)

type MeetingRepository struct {
	db *bun.DB
}

func NewMeetingRepository(db *bun.DB) *MeetingRepository {
	return &MeetingRepository{db: db}
}

func (r *MeetingRepository) CreateMeeting(ctx context.Context, req *model.Meeting) (*model.Meeting, error) {
	_, err := r.db.NewInsert().Model(req).Exec(ctx)
	if err != nil {
		return nil, err
	}

	return req, nil
}

func (r *MeetingRepository) GetMeetingById(ctx context.Context, id uint32) (*model.Meeting, error) {
	var meeting model.Meeting
	err := r.db.NewSelect().Model(&meeting).Where("id = ?", id).Scan(ctx)
	if err != nil {
		return nil, err
	}

	return &meeting, nil
}

func (r *MeetingRepository) GetMeetings(ctx context.Context, cursor string, limit uint32) ([]*model.Meeting, string, error) {
	var meetings []*model.Meeting

	query := r.db.NewSelect().Model(&meetings).Order("id DESC").Limit(int(limit) + 1)

	if cursor != "" {
		query.Where("id < ?", cursor)
	}

	err := query.Scan(ctx)
	if err != nil {
		return nil, "", err
	}

	var nextCursor string
	if len(meetings) > int(limit) {
		nextCursor = fmt.Sprint(meetings[limit-1].ID)
		meetings = meetings[:limit]
	}

	return meetings, nextCursor, nil
}

func (r *MeetingRepository) UpdateMeeting(ctx context.Context, meeting *model.Meeting) (*model.Meeting, error) {
	_, err := r.db.NewUpdate().Model(meeting).Where("id = ?", meeting.ID).Exec(ctx)
	if err != nil {
		return nil, err
	}

	return meeting, nil
}

func (r *MeetingRepository) DeleteMeeting(ctx context.Context, meetingId uint32) error {
	_, err := r.db.NewDelete().Model((*model.Meeting)(nil)).Where("id = ?", meetingId).Exec(ctx)
	return err
}

/*
MeetingWithVotes 조회
*/
func (r *MeetingRepository) GetMeetingWithVotesByInviteCode(ctx context.Context, inviteCode string) (*model.MeetingWithVotes, error) {
	var meeting model.Meeting
	err := r.db.NewSelect().Model(&meeting).Where("invite_code = ?", inviteCode).Scan(ctx)
	if err != nil {
		return nil, err
	}

	var votes []*model.Vote
	err = r.db.NewSelect().Model(&votes).Where("meeting_id = ?", meeting.ID).Scan(ctx)
	if err != nil {
		return nil, err
	}

	return &model.MeetingWithVotes{
		Meeting: &meeting,
		Votes:   votes,
	}, nil
}

func (r *MeetingRepository) GetMeetingByInviteCode(ctx context.Context, inviteCode string) (*model.Meeting, error) {
	var meeting model.Meeting
	err := r.db.NewSelect().Model(&meeting).Where("invite_code = ?", inviteCode).Scan(ctx)
	if err != nil {
		return nil, err
	}

	return &meeting, nil
}

func (r *MeetingRepository) GetMeetingByShortId(ctx context.Context, shortId string) (*model.Meeting, error) {
	var meeting model.Meeting
	err := r.db.NewSelect().Model(&meeting).Where("short_id = ?", shortId).Scan(ctx)
	if err != nil {
		return nil, err
	}

	return &meeting, nil
}

func (r *MeetingRepository) GetVotesByMeetingId(ctx context.Context, meetingId uint32) ([]*model.Vote, error) {
	var votes []*model.Vote
	err := r.db.NewSelect().Model(&votes).Where("meeting_id = ?", meetingId).Scan(ctx)
	if err != nil {
		return nil, err
	}

	return votes, nil
}

func (r *MeetingRepository) FindPendingJoinRequest(ctx context.Context, meetingId uint32, requesterId string) (*model.JoinRequest, error) {
	var req model.JoinRequest
	err := r.db.NewSelect().
		Model(&req).
		Where("meeting_id = ?", meetingId).
		Where("requester_id = ?", requesterId).
		Where("status = ?", "pending").
		Order("id DESC").
		Limit(1).
		Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &req, nil
}

func (r *MeetingRepository) CreateJoinRequest(ctx context.Context, req *model.JoinRequest) (*model.JoinRequest, error) {
	_, err := r.db.NewInsert().Model(req).Exec(ctx)
	if err != nil {
		return nil, err
	}

	return req, nil
}

func (r *MeetingRepository) GetJoinRequestsByMeetingID(ctx context.Context, meetingId uint32) ([]*model.JoinRequest, error) {
	var requests []*model.JoinRequest
	err := r.db.NewSelect().
		Model(&requests).
		Where("meeting_id = ?", meetingId).
		Order("created_at DESC").
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	return requests, nil
}

func (r *MeetingRepository) GetJoinRequestByID(ctx context.Context, requestId uint32) (*model.JoinRequest, error) {
	var request model.JoinRequest
	err := r.db.NewSelect().Model(&request).Where("id = ?", requestId).Scan(ctx)
	if err != nil {
		return nil, err
	}

	return &request, nil
}

func (r *MeetingRepository) UpdateJoinRequest(ctx context.Context, request *model.JoinRequest) error {
	_, err := r.db.NewUpdate().
		Model(request).
		Column("status", "participant_code", "processed_by").
		Set("updated_at = NOW()").
		WherePK().
		Exec(ctx)
	return err
}

func (r *MeetingRepository) UpsertApprovedParticipant(
	ctx context.Context,
	meetingId uint32,
	requesterId string,
	requesterName string,
	participantCode string,
	approvedBy string,
) (*model.MeetingParticipant, error) {
	participant := &model.MeetingParticipant{
		MeetingId:       meetingId,
		RequesterId:     requesterId,
		RequesterName:   requesterName,
		ParticipantCode: participantCode,
		Status:          "active",
		ApprovedBy:      approvedBy,
	}

	_, err := r.db.NewInsert().
		Model(participant).
		On("CONFLICT (meeting_id, requester_id) DO UPDATE").
		Set("requester_name = EXCLUDED.requester_name").
		Set("participant_code = EXCLUDED.participant_code").
		Set("status = 'active'").
		Set("approved_by = EXCLUDED.approved_by").
		Set("updated_at = NOW()").
		Returning("*").
		Exec(ctx)
	if err != nil {
		return nil, err
	}

	return participant, nil
}

func (r *MeetingRepository) GetActiveParticipantByCode(ctx context.Context, meetingId uint32, participantCode string) (*model.MeetingParticipant, error) {
	var participant model.MeetingParticipant
	err := r.db.NewSelect().
		Model(&participant).
		Where("meeting_id = ?", meetingId).
		Where("participant_code = ?", participantCode).
		Where("status = ?", "active").
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	return &participant, nil
}

func (r *MeetingRepository) SubmitVotes(ctx context.Context,
	meetingId uint32,
	selectedSlots []string,
	participantKey string,
	voterName string) error {
	if voterName == "" {
		voterName = participantKey
	}

	if len(selectedSlots) == 0 {
		_, err := r.db.NewDelete().Model((*model.Vote)(nil)).
			Where("meeting_id = ?", meetingId).
			Where("user_id = ?", participantKey).
			Exec(ctx)
		return err
	}

	vote := &model.Vote{
		MeetingId: meetingId,
		UserId:    participantKey,
		UserName:  voterName,
		Slots:     selectedSlots,
	}

	_, err := r.db.NewInsert().Model(vote).
		On("CONFLICT (meeting_id, user_id) DO UPDATE").
		Set("user_name = EXCLUDED.user_name").
		Set("slots = EXCLUDED.slots").
		Set("updated_at = NOW()").
		Exec(ctx)
	return err
}

func (r *MeetingRepository) DeleteVotes(ctx context.Context, meetingId uint32, participantCode string) error {
	_, err := r.db.NewDelete().Model((*model.Vote)(nil)).
		Where("meeting_id = ?", meetingId).
		Where("user_id = ?", participantCode).
		Exec(ctx)
	return err
}

func (r *MeetingRepository) FinalizeMeeting(
	ctx context.Context,
	meetingId uint32,
	slot string,
	finalizedBy string,
) error {
	_, err := r.db.NewUpdate().
		Model((*model.Meeting)(nil)).
		Set("final_slot = ?", slot).
		Set("finalized_by = ?", finalizedBy).
		Set("finalized_at = NOW()").
		Set("updated_at = NOW()").
		Where("id = ?", meetingId).
		Exec(ctx)
	return err
}

func (r *MeetingRepository) ClearMeetingFinalization(
	ctx context.Context,
	meetingId uint32,
) error {
	_, err := r.db.NewUpdate().
		Model((*model.Meeting)(nil)).
		Set("final_slot = NULL").
		Set("finalized_by = NULL").
		Set("finalized_at = NULL").
		Set("updated_at = NOW()").
		Where("id = ?", meetingId).
		Exec(ctx)
	return err
}

func (r *MeetingRepository) AddPushSubscription(
	ctx context.Context,
	meetingId uint32,
	userId string,
	deviceId string,
	isStandalone bool,
	notificationPermissionStatus string,
	endpoint string,
	auth string,
	p256dh string,
) error {
	record := &model.NotificationSubscription{
		MeetingId:                    meetingId,
		UserId:                       userId,
		DeviceId:                     deviceId,
		Endpoint:                     endpoint,
		P256dh:                       p256dh,
		Auth:                         auth,
		IsStandalone:                 isStandalone,
		NotificationPermissionStatus: notificationPermissionStatus,
		IsActive:                     true,
		EndpointStatus:               "active",
	}

	_, err := r.db.NewInsert().
		Model(record).
		On("CONFLICT (meeting_id, user_id, device_id) DO UPDATE").
		Set("endpoint = EXCLUDED.endpoint").
		Set("p256dh = EXCLUDED.p256dh").
		Set("auth = EXCLUDED.auth").
		Set("is_standalone = EXCLUDED.is_standalone").
		Set("notification_permission_status = EXCLUDED.notification_permission_status").
		Set("is_active = TRUE").
		Set("endpoint_status = 'active'").
		Set("last_verified_at = NOW()").
		Set("updated_at = NOW()").
		Exec(ctx)

	return err
}

func (r *MeetingRepository) RemovePushSubscription(ctx context.Context, meetingId uint32, userId string, deviceId string) error {
	_, err := r.db.NewUpdate().
		Model((*model.NotificationSubscription)(nil)).
		Set("is_active = FALSE").
		Set("updated_at = NOW()").
		Where("meeting_id = ?", meetingId).
		Where("user_id = ?", userId).
		Where("device_id = ?", deviceId).
		Exec(ctx)

	return err
}

func (r *MeetingRepository) GetPushSubscriptionByDevice(ctx context.Context, meetingId uint32, userId string, deviceId string) (*model.NotificationSubscription, error) {
	var sub model.NotificationSubscription
	err := r.db.NewSelect().
		Model(&sub).
		Where("meeting_id = ?", meetingId).
		Where("user_id = ?", userId).
		Where("device_id = ?", deviceId).
		Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &sub, nil
}

func (r *MeetingRepository) GetPushSubscriptionsByUser(ctx context.Context, meetingId uint32, userId string) ([]*model.NotificationSubscription, error) {
	var subs []*model.NotificationSubscription
	err := r.db.NewSelect().
		Model(&subs).
		Where("meeting_id = ?", meetingId).
		Where("user_id = ?", userId).
		Order("updated_at DESC").
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	return subs, nil
}

func (r *MeetingRepository) CountAttendanceReminderTargets(ctx context.Context, meetingId uint32) (int, error) {
	var count int
	err := r.db.NewSelect().
		TableExpr("meeting_participants AS mp").
		ColumnExpr("COUNT(DISTINCT mp.requester_id)").
		Join("LEFT JOIN votes AS v ON v.meeting_id = mp.meeting_id AND v.user_id = mp.requester_id").
		Join("JOIN notification_subscriptions AS ns ON ns.meeting_id = mp.meeting_id AND ns.user_id = mp.requester_id").
		Where("mp.meeting_id = ?", meetingId).
		Where("mp.status = ?", "active").
		Where("v.id IS NULL").
		Where("ns.is_active = TRUE").
		Where("ns.endpoint_status = ?", "active").
		Where("ns.notification_permission_status = ?", "granted").
		Scan(ctx, &count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (r *MeetingRepository) CreateAttendanceNudge(ctx context.Context, nudge *model.AttendanceNudge) (*model.AttendanceNudge, error) {
	_, err := r.db.NewInsert().
		Model(nudge).
		Returning("*").
		Exec(ctx)
	if err != nil {
		return nil, err
	}

	return nudge, nil
}
