package service

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/big"
	"meetBack/internal/model"
	"meetBack/internal/repository"
	"strings"
	"time"
)

type MeetingService struct {
	repository *repository.MeetingRepository
	pushSender PushSender
}

type PushSender interface {
	Send(
		ctx context.Context,
		subscription *model.NotificationSubscription,
		payload []byte,
	) error
}

type CreateMeetingInput struct {
	Title       string
	Description string
	Location    string
	StartTime   string
	EndTime     string
	Dates       []string
	JoinPolicy  string
	HostId      string
	HostName    string
}

type UpdateMeetingInput struct {
	Title       string
	Description string
	Location    string
	StartTime   string
	EndTime     string
	Dates       []string
	HostId      string
	HostName    string
}

type JoinRequestCreateInput struct {
	InviteCode    string
	RequesterId   string
	RequesterName string
}

type JoinRequestCreateResult struct {
	MeetingId       uint32
	Status          string
	RequestId       *uint32
	ParticipantCode string
}

type JoinRequestResolveResult struct {
	Request         *model.JoinRequest
	ParticipantCode string
}

type FinalizeMeetingInput struct {
	Slot string
}

type MeetingFinalResult struct {
	MeetingId   uint32
	Slot        string
	FinalizedBy string
	FinalizedAt time.Time
}

type MeetingStatusResult struct {
	MeetingId uint32
	IsClosed  bool
	ClosedAt  *time.Time
}

var (
	ErrForbidden    = errors.New("forbidden")
	ErrNotFound     = errors.New("not found")
	ErrInvalidState = errors.New("invalid state")
	ErrInvalidInput = errors.New("invalid input")
)

func NewMeetingService(repository *repository.MeetingRepository, pushSender PushSender) *MeetingService {
	return &MeetingService{repository: repository, pushSender: pushSender}
}

func (s *MeetingService) CreateMeeting(ctx context.Context, in CreateMeetingInput) (*model.Meeting, error) {
	// startTime <= endTime 검증 및 날짜 형식 검증
	startMinutes, err := parseTimeMinutes(in.StartTime, false)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid start time: %v", ErrInvalidInput, err)
	}
	endMinutes, err := parseTimeMinutes(in.EndTime, true)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid end time: %v", ErrInvalidInput, err)
	}
	if startMinutes >= endMinutes {
		return nil, fmt.Errorf("%w: start time must be before end time", ErrInvalidInput)
	}

	parsedDates := make([]time.Time, 0, len(in.Dates))
	for _, d := range in.Dates {
		t, err := time.Parse("2006-01-02", d)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid date: %s", ErrInvalidInput, d)
		}
		parsedDates = append(parsedDates, t)
	}

	meeting := &model.Meeting{
		Title:        in.Title,
		Description:  in.Description,
		Location:     in.Location,
		StartTime:    in.StartTime,
		EndTime:      in.EndTime,
		Dates:        parsedDates,
		ShortId:      randomAlphaNum(10),
		InviteCode:   randomInviteCode(),
		InvitePolicy: normalizeJoinPolicy(in.JoinPolicy),
		HostId:       in.HostId,
		HostName:     in.HostName,
	}

	return s.repository.CreateMeeting(ctx, meeting)
}

func (s *MeetingService) UpdateMeeting(ctx context.Context, meetingId uint32, in UpdateMeetingInput) (*model.Meeting, error) {
	// meetingId로 미팅 존재 여부 확인
	existing, err := s.repository.GetMeetingById(ctx, meetingId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	// 주최자 검증 (HostId 비교)
	if existing.HostId != in.HostId {
		return nil, ErrForbidden
	}

	// 업데이트할 필드만 업데이트
	if in.Title != "" {
		existing.Title = in.Title
	}
	if in.Description != "" {
		existing.Description = in.Description
	}
	if in.Location != "" {
		existing.Location = in.Location
	}

	startTime := existing.StartTime
	if in.StartTime != "" {
		startTime = in.StartTime
	}
	endTime := existing.EndTime
	if in.EndTime != "" {
		endTime = in.EndTime
	}

	startMinutes, err := parseTimeMinutes(startTime, false)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid start time: %v", ErrInvalidInput, err)
	}
	endMinutes, err := parseTimeMinutes(endTime, true)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid end time: %v", ErrInvalidInput, err)
	}
	if startMinutes >= endMinutes {
		return nil, fmt.Errorf("%w: start time must be before end time", ErrInvalidInput)
	}
	existing.StartTime = startTime
	existing.EndTime = endTime

	var parsedDates []time.Time

	if len(in.Dates) > 0 {
		for _, d := range in.Dates {
			t, err := time.Parse("2006-01-02", d)
			if err != nil {
				return nil, fmt.Errorf("%w: invalid date: %s", ErrInvalidInput, d)
			}
			parsedDates = append(parsedDates, t)
		}
		existing.Dates = parsedDates
	}

	return s.repository.UpdateMeeting(ctx, existing)
}

func (s *MeetingService) DeleteMeeting(ctx context.Context, meetingId uint32, hostId string) error {
	// meetingId로 미팅 존재 여부 확인
	existing, err := s.repository.GetMeetingById(ctx, meetingId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}

	// 주최자 검증 (HostId 비교)
	if existing.HostId != hostId {
		return ErrForbidden
	}

	return s.repository.DeleteMeeting(ctx, meetingId)
}

func (s *MeetingService) GetMeetings(ctx context.Context, userId string, cursor string, limit uint32) ([]*model.Meeting, string, error) {
	return s.repository.GetMeetings(ctx, userId, cursor, limit)
}

func (s *MeetingService) GetMeetingById(ctx context.Context, meetingId uint32) (*model.Meeting, error) {
	meeting, err := s.repository.GetMeetingById(ctx, meetingId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return meeting, nil
}

var alphaNum = []rune("abcdefghijklmnopqrstuvwxyz0123456789")
var upperNum = []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

func randomAlphaNum(n int) string {
	out := make([]rune, n)
	for i := 0; i < n; i++ {
		v, _ := rand.Int(rand.Reader, big.NewInt(int64(len(alphaNum))))
		out[i] = alphaNum[v.Int64()]
	}
	return string(out)
}

func randomInviteCode() string {
	left := make([]rune, 3)
	right := make([]rune, 4)
	for i := 0; i < 3; i++ {
		v, _ := rand.Int(rand.Reader, big.NewInt(int64(len(upperNum))))
		left[i] = upperNum[v.Int64()]
	}
	for i := 0; i < 4; i++ {
		v, _ := rand.Int(rand.Reader, big.NewInt(int64(len(upperNum))))
		right[i] = upperNum[v.Int64()]
	}
	return string(left) + "-" + string(right)
}

func (s *MeetingService) GetMeetingByInviteCode(ctx context.Context, inviteCode string) (*model.Meeting, error) {
	meeting, err := s.repository.GetMeetingByInviteCode(ctx, inviteCode)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return meeting, nil
}

func (s *MeetingService) GetMeetingByShortId(ctx context.Context, shortId string) (*model.Meeting, error) {
	meeting, err := s.repository.GetMeetingByShortId(ctx, shortId)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return meeting, nil
}

func (s *MeetingService) GetVotesByMeetingId(ctx context.Context, meetingId uint32) ([]*model.Vote, error) {
	votes, err := s.repository.GetVotesByMeetingId(ctx, meetingId)
	if err != nil {
		return nil, err
	}

	return votes, nil
}

func (s *MeetingService) GetVotesByMeetingIdAuthorized(ctx context.Context, meetingId uint32, participantCode string, hostId string) ([]*model.Vote, error) {
	meeting, err := s.repository.GetMeetingById(ctx, meetingId)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("[votes:auth] meetingId=%d not_found", meetingId)
			return nil, ErrNotFound
		}
		log.Printf("[votes:auth] meetingId=%d load_meeting_failed err=%v", meetingId, err)
		return nil, err
	}

	if hostId != "" && hostId == meeting.HostId {
		log.Printf("[votes:auth] meetingId=%d access=host", meetingId)
		return s.repository.GetVotesByMeetingId(ctx, meetingId)
	}

	hostId = strings.TrimSpace(hostId)
	if hostId != "" {
		hasVote, err := s.repository.HasVoteByMeetingAndUser(ctx, meetingId, hostId)
		if err != nil {
			log.Printf("[votes:auth] meetingId=%d check_user_vote_failed err=%v", meetingId, err)
			return nil, err
		}
		if hasVote {
			log.Printf("[votes:auth] meetingId=%d access=participant_user", meetingId)
			return s.repository.GetVotesByMeetingId(ctx, meetingId)
		}
	}

	participantCode = strings.TrimSpace(participantCode)
	if participantCode == "" {
		log.Printf(
			"[votes:auth] meetingId=%d access=denied reason=missing_participant_code host_id_set=%t",
			meetingId,
			strings.TrimSpace(hostId) != "",
		)
		return nil, ErrForbidden
	}

	if strings.HasPrefix(participantCode, "guest:") {
		hasVote, err := s.repository.HasVoteByMeetingAndUser(ctx, meetingId, participantCode)
		if err != nil {
			log.Printf("[votes:auth] meetingId=%d check_guest_vote_failed err=%v", meetingId, err)
			return nil, err
		}
		if !hasVote {
			log.Printf("[votes:auth] meetingId=%d access=denied reason=guest_vote_not_found", meetingId)
			return nil, ErrForbidden
		}
		log.Printf("[votes:auth] meetingId=%d access=guest_participant_code", meetingId)
		return s.repository.GetVotesByMeetingId(ctx, meetingId)
	}

	_, valid, err := s.VerifyParticipantCode(ctx, meetingId, participantCode)
	if err != nil {
		log.Printf("[votes:auth] meetingId=%d verify_participant_code_failed err=%v", meetingId, err)
		return nil, err
	}
	if !valid {
		log.Printf("[votes:auth] meetingId=%d access=denied reason=invalid_participant_code", meetingId)
		return nil, ErrForbidden
	}
	log.Printf("[votes:auth] meetingId=%d access=participant_code", meetingId)

	return s.repository.GetVotesByMeetingId(ctx, meetingId)
}

func (s *MeetingService) CreateJoinRequestByInviteCode(ctx context.Context, in JoinRequestCreateInput) (*JoinRequestCreateResult, error) {
	requesterId := strings.TrimSpace(in.RequesterId)
	requesterName := strings.TrimSpace(in.RequesterName)
	if requesterId == "" || requesterName == "" {
		return nil, fmt.Errorf("%w: requester id/name are required", ErrInvalidInput)
	}

	meeting, err := s.repository.GetMeetingByInviteCode(ctx, strings.TrimSpace(in.InviteCode))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}

	if normalizeJoinPolicy(meeting.InvitePolicy) == "approval" {
		existingPending, err := s.repository.FindPendingJoinRequest(ctx, meeting.ID, requesterId)
		if err != nil {
			return nil, err
		}

		if existingPending != nil {
			requestId := existingPending.ID
			return &JoinRequestCreateResult{
				MeetingId: meeting.ID,
				Status:    "pending",
				RequestId: &requestId,
			}, nil
		}

		created, err := s.repository.CreateJoinRequest(ctx, &model.JoinRequest{
			MeetingId:     meeting.ID,
			RequesterId:   requesterId,
			RequesterName: requesterName,
			Status:        "pending",
		})
		if err != nil {
			return nil, err
		}

		requestId := created.ID
		return &JoinRequestCreateResult{
			MeetingId: meeting.ID,
			Status:    "pending",
			RequestId: &requestId,
		}, nil
	}

	participant, err := s.repository.UpsertApprovedParticipant(
		ctx,
		meeting.ID,
		requesterId,
		requesterName,
		randomParticipantCode(),
		"system-auto",
	)
	if err != nil {
		return nil, err
	}

	return &JoinRequestCreateResult{
		MeetingId:       meeting.ID,
		Status:          "approved",
		ParticipantCode: participant.ParticipantCode,
	}, nil
}

func (s *MeetingService) GetJoinRequestsForHost(ctx context.Context, meetingId uint32, hostId string) ([]*model.JoinRequest, error) {
	if _, err := s.ensureHostOfMeeting(ctx, meetingId, hostId); err != nil {
		return nil, err
	}

	return s.repository.GetJoinRequestsByMeetingID(ctx, meetingId)
}

func (s *MeetingService) ApproveJoinRequest(ctx context.Context, meetingId uint32, requestId uint32, hostId string) (*JoinRequestResolveResult, error) {
	if _, err := s.ensureHostOfMeeting(ctx, meetingId, hostId); err != nil {
		return nil, err
	}

	request, err := s.repository.GetJoinRequestByID(ctx, requestId)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}

	if request.MeetingId != meetingId {
		return nil, ErrNotFound
	}

	if request.Status != "pending" {
		return nil, ErrInvalidState
	}

	participantCode := request.ParticipantCode
	if strings.TrimSpace(participantCode) == "" {
		participantCode = randomParticipantCode()
	}

	participant, err := s.repository.UpsertApprovedParticipant(
		ctx,
		meetingId,
		request.RequesterId,
		request.RequesterName,
		participantCode,
		hostId,
	)
	if err != nil {
		return nil, err
	}

	request.Status = "approved"
	request.ProcessedBy = hostId
	request.ParticipantCode = participant.ParticipantCode
	if err := s.repository.UpdateJoinRequest(ctx, request); err != nil {
		return nil, err
	}

	return &JoinRequestResolveResult{
		Request:         request,
		ParticipantCode: participant.ParticipantCode,
	}, nil
}

func (s *MeetingService) RejectJoinRequest(ctx context.Context, meetingId uint32, requestId uint32, hostId string) (*model.JoinRequest, error) {
	if _, err := s.ensureHostOfMeeting(ctx, meetingId, hostId); err != nil {
		return nil, err
	}

	request, err := s.repository.GetJoinRequestByID(ctx, requestId)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}

	if request.MeetingId != meetingId {
		return nil, ErrNotFound
	}

	if request.Status != "pending" {
		return nil, ErrInvalidState
	}

	request.Status = "rejected"
	request.ProcessedBy = hostId
	request.ParticipantCode = ""
	if err := s.repository.UpdateJoinRequest(ctx, request); err != nil {
		return nil, err
	}

	return request, nil
}

func (s *MeetingService) VerifyParticipantCode(ctx context.Context, meetingId uint32, participantCode string) (*model.MeetingParticipant, bool, error) {
	participantCode = strings.TrimSpace(participantCode)
	if participantCode == "" {
		return nil, false, nil
	}

	participant, err := s.repository.GetActiveParticipantByCode(ctx, meetingId, participantCode)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, false, nil
		}
		return nil, false, err
	}

	return participant, true, nil
}

func (s *MeetingService) ensureHostOfMeeting(ctx context.Context, meetingId uint32, hostId string) (*model.Meeting, error) {
	meeting, err := s.repository.GetMeetingById(ctx, meetingId)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}

	if meeting.HostId != strings.TrimSpace(hostId) {
		return nil, ErrForbidden
	}

	return meeting, nil
}

func normalizeJoinPolicy(policy string) string {
	if strings.EqualFold(strings.TrimSpace(policy), "approval") {
		return "approval"
	}

	return "auto"
}

func randomParticipantCode() string {
	out := make([]rune, 10)
	for i := 0; i < len(out); i++ {
		v, _ := rand.Int(rand.Reader, big.NewInt(int64(len(upperNum))))
		out[i] = upperNum[v.Int64()]
	}
	return string(out)
}

func parseTimeMinutes(value string, allow24 bool) (int, error) {
	if value == "24:00" {
		if allow24 {
			return 24 * 60, nil
		}
		return 0, fmt.Errorf("24:00 is not allowed")
	}

	t, err := time.Parse("15:04", value)
	if err != nil {
		return 0, err
	}

	return t.Hour()*60 + t.Minute(), nil
}

func (s *MeetingService) SubmitVotesRequest(meetingId uint32, selectedSlots []string, participantCode string, hostId string, hostName string) error {
	meeting, err := s.repository.GetMeetingById(context.Background(), meetingId)
	if err != nil {
		if err == sql.ErrNoRows {
			return ErrNotFound
		}
		return err
	}

	participantCode = strings.TrimSpace(participantCode)
	hostId = strings.TrimSpace(hostId)
	hostName = strings.TrimSpace(hostName)

	if hostId != "" && hostId == meeting.HostId {
		participantKey := participantCode
		if participantKey == "" {
			participantKey = hostId
		}
		return s.repository.SubmitVotes(context.Background(), meetingId, selectedSlots, participantKey, hostName)
	}

	// participantCode가 있으면 기존 승인 코드 경로를 우선 사용
	if participantCode != "" {
		if strings.HasPrefix(participantCode, "guest:") {
			if hostName == "" {
				hostName = "Guest"
			}
			return s.repository.SubmitVotes(context.Background(), meetingId, selectedSlots, participantCode, hostName)
		}

		_, valid, err := s.VerifyParticipantCode(context.Background(), meetingId, participantCode)
		if err != nil {
			return err
		}
		if !valid {
			return ErrForbidden
		}
		return s.repository.SubmitVotes(context.Background(), meetingId, selectedSlots, participantCode, hostName)
	}

	// 로그인 사용자라면 participantCode 없이도 userId 기준으로 투표 저장 허용
	if hostId == "" {
		return ErrForbidden
	}
	return s.repository.SubmitVotes(context.Background(), meetingId, selectedSlots, hostId, hostName)
}

func (s *MeetingService) DeleteVotesRequest(meetingId uint32, participantCode string, hostId string) error {
	meeting, err := s.repository.GetMeetingById(context.Background(), meetingId)
	if err != nil {
		if err == sql.ErrNoRows {
			return ErrNotFound
		}
		return err
	}

	if hostId != "" && hostId == meeting.HostId {
		return s.repository.DeleteVotes(context.Background(), meetingId, participantCode)
	}

	participantCode = strings.TrimSpace(participantCode)
	if participantCode == "" {
		return ErrForbidden
	}

	_, valid, err := s.VerifyParticipantCode(context.Background(), meetingId, participantCode)
	if err != nil {
		return err
	}
	if !valid {
		return ErrForbidden
	}

	return s.repository.DeleteVotes(context.Background(), meetingId, participantCode)
}

func (s *MeetingService) FinalizeMeeting(
	ctx context.Context,
	meetingId uint32,
	hostId string,
	in FinalizeMeetingInput,
) (*MeetingFinalResult, error) {
	meeting, err := s.ensureHostOfMeeting(ctx, meetingId, hostId)
	if err != nil {
		return nil, err
	}

	slot := strings.TrimSpace(in.Slot)
	if slot == "" {
		return nil, fmt.Errorf("%w: slot is required", ErrInvalidInput)
	}

	if err := validateFinalSlot(slot, meeting); err != nil {
		return nil, err
	}

	finalizedBy := strings.TrimSpace(meeting.HostName)
	if finalizedBy == "" {
		finalizedBy = strings.TrimSpace(hostId)
	}

	if err := s.repository.FinalizeMeeting(ctx, meetingId, slot, finalizedBy); err != nil {
		return nil, err
	}

	updated, err := s.repository.GetMeetingById(ctx, meetingId)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}

	if updated.FinalizedAt == nil {
		return nil, fmt.Errorf("%w: finalized timestamp missing", ErrInvalidState)
	}

	return &MeetingFinalResult{
		MeetingId:   updated.ID,
		Slot:        updated.FinalSlot,
		FinalizedBy: updated.FinalizedBy,
		FinalizedAt: *updated.FinalizedAt,
	}, nil
}

func (s *MeetingService) GetMeetingFinal(
	ctx context.Context,
	meetingId uint32,
) (*MeetingFinalResult, error) {
	meeting, err := s.repository.GetMeetingById(ctx, meetingId)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}

	if strings.TrimSpace(meeting.FinalSlot) == "" || meeting.FinalizedAt == nil {
		return nil, ErrNotFound
	}

	return &MeetingFinalResult{
		MeetingId:   meeting.ID,
		Slot:        meeting.FinalSlot,
		FinalizedBy: meeting.FinalizedBy,
		FinalizedAt: *meeting.FinalizedAt,
	}, nil
}

func (s *MeetingService) ClearMeetingFinal(
	ctx context.Context,
	meetingId uint32,
	hostId string,
) error {
	if _, err := s.ensureHostOfMeeting(ctx, meetingId, hostId); err != nil {
		return err
	}

	return s.repository.ClearMeetingFinalization(ctx, meetingId)
}

func (s *MeetingService) SetMeetingClosed(
	ctx context.Context,
	meetingId uint32,
	hostId string,
	isClosed bool,
) (*MeetingStatusResult, error) {
	if _, err := s.ensureHostOfMeeting(ctx, meetingId, hostId); err != nil {
		return nil, err
	}

	if err := s.repository.SetMeetingClosed(ctx, meetingId, isClosed); err != nil {
		return nil, err
	}

	updated, err := s.repository.GetMeetingById(ctx, meetingId)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &MeetingStatusResult{
		MeetingId: updated.ID,
		IsClosed:  updated.IsClosed,
		ClosedAt:  updated.ClosedAt,
	}, nil
}

func validateFinalSlot(slot string, meeting *model.Meeting) error {
	if len(slot) != 16 || slot[10] != '-' {
		return fmt.Errorf("%w: slot format must be YYYY-MM-DD-HH:mm", ErrInvalidInput)
	}

	datePart := slot[:10]
	timePart := slot[11:]

	if _, err := time.Parse("2006-01-02", datePart); err != nil {
		return fmt.Errorf("%w: invalid slot date", ErrInvalidInput)
	}

	timeMin, err := parseTimeMinutes(timePart, false)
	if err != nil {
		return fmt.Errorf("%w: invalid slot time", ErrInvalidInput)
	}
	if timeMin%30 != 0 {
		return fmt.Errorf("%w: slot must align to 30-minute intervals", ErrInvalidInput)
	}

	startMin, err := parseTimeMinutes(meeting.StartTime, false)
	if err != nil {
		return fmt.Errorf("%w: meeting start time invalid", ErrInvalidState)
	}
	endMin, err := parseTimeMinutes(meeting.EndTime, true)
	if err != nil {
		return fmt.Errorf("%w: meeting end time invalid", ErrInvalidState)
	}
	if timeMin < startMin || timeMin >= endMin {
		return fmt.Errorf("%w: slot is outside meeting time range", ErrInvalidInput)
	}

	dateAllowed := false
	for _, d := range meeting.Dates {
		if d.Format("2006-01-02") == datePart {
			dateAllowed = true
			break
		}
	}
	if !dateAllowed {
		return fmt.Errorf("%w: slot date not in candidate dates", ErrInvalidInput)
	}

	return nil
}

type AddPushSubscriptionInput struct {
	DeviceId                     string                `json:"deviceId" binding:"required,min=8,max=64"`
	IsStandalone                 bool                  `json:"isStandalone"`
	NotificationPermissionStatus string                `json:"notificationPermissionStatus" binding:"required,oneof=granted denied default"`
	PushSubscription             PushSubscriptionInput `json:"pushSubscription" binding:"required"`
}

type PushSubscriptionInput struct {
	Endpoint string                    `json:"endpoint" binding:"required"`
	Keys     PushSubscriptionKeysInput `json:"keys" binding:"required"`
}

type PushSubscriptionKeysInput struct {
	Auth   string `json:"auth" binding:"required"`
	P256dh string `json:"p256dh" binding:"required"`
}

type PushSubscriptionStatusResponse struct {
	MeetingId                      uint32     `json:"meetingId"`
	UserId                         string     `json:"userId"`
	DeviceId                       string     `json:"deviceId"`
	IsSubscribed                   bool       `json:"isSubscribed"`
	IsStandalone                   bool       `json:"isStandalone"`
	NotificationPermissionStatus   string     `json:"notificationPermissionStatus"`
	InstallFlagStatus              string     `json:"installFlagStatus"`
	PushSubscriptionEndpointStatus string     `json:"pushSubscriptionEndpointStatus"`
	LastVerifiedAt                 *time.Time `json:"lastVerifiedAt,omitempty"`
	LastNudgeAt                    *time.Time `json:"lastNudgeAt"`
}

type SendAttendanceReminderResponse struct {
	NudgeId     string    `json:"nudgeId"`
	MeetingId   uint32    `json:"meetingId"`
	TargetCount int       `json:"targetCount"`
	QueuedAt    time.Time `json:"queuedAt"`
}

type SendTestPushInput struct {
	Title string
	Body  string
	URL   string
	Tag   string
}

type SendTestPushResult struct {
	DeviceId string `json:"deviceId"`
	Success  bool   `json:"success"`
	Error    string `json:"error,omitempty"`
}

type SendTestPushResponse struct {
	MeetingId   uint32               `json:"meetingId"`
	SentCount   int                  `json:"sentCount"`
	FailCount   int                  `json:"failCount"`
	Results     []SendTestPushResult `json:"results"`
	TriggeredAt time.Time            `json:"triggeredAt"`
}

func (s *MeetingService) SendAttendanceReminder(ctx context.Context, meetingId uint32, hostId string, messageOverride string) (*SendAttendanceReminderResponse, error) {
	if _, err := s.ensureHostOfMeeting(ctx, meetingId, hostId); err != nil {
		return nil, err
	}

	if s.pushSender == nil {
		return nil, fmt.Errorf("%w: push sender is not configured", ErrInvalidState)
	}

	trimmedMessageOverride := strings.TrimSpace(messageOverride)
	if len(trimmedMessageOverride) > 200 {
		return nil, fmt.Errorf("%w: messageOverride must be 200 characters or fewer", ErrInvalidInput)
	}

	targetCount, err := s.repository.CountAttendanceReminderTargets(ctx, meetingId)
	if err != nil {
		return nil, err
	}

	if targetCount == 0 {
		return nil, fmt.Errorf("%w: no pending participants to remind", ErrInvalidState)
	}

	nudge, err := s.repository.CreateAttendanceNudge(ctx, &model.AttendanceNudge{
		NudgeId:           "nudge_" + randomAlphaNum(12),
		MeetingId:         meetingId,
		TriggerType:       "manual",
		RequestedByUserId: strings.TrimSpace(hostId),
		MessageOverride:   trimmedMessageOverride,
		TargetCount:       targetCount,
		Status:            "queued",
	})
	if err != nil {
		return nil, err
	}

	return &SendAttendanceReminderResponse{
		NudgeId:     nudge.NudgeId,
		MeetingId:   meetingId,
		TargetCount: targetCount,
		QueuedAt:    nudge.QueuedAt,
	}, nil
}

func (s *MeetingService) AddPushSubscription(ctx context.Context, meetingId uint32, userId string, input AddPushSubscriptionInput) error {
	meeting, err := s.repository.GetMeetingById(ctx, meetingId)
	if err != nil {
		if err == sql.ErrNoRows {
			return ErrNotFound
		}
		return err
	}

	if strings.TrimSpace(userId) == "" {
		return ErrForbidden
	}

	if !input.IsStandalone {
		return fmt.Errorf("%w: standalone mode required", ErrInvalidInput)
	}

	if input.NotificationPermissionStatus != "granted" {
		return fmt.Errorf("%w: notification permission must be granted", ErrInvalidInput)
	}

	if strings.TrimSpace(input.PushSubscription.Endpoint) == "" ||
		strings.TrimSpace(input.PushSubscription.Keys.Auth) == "" ||
		strings.TrimSpace(input.PushSubscription.Keys.P256dh) == "" {
		return fmt.Errorf("%w: invalid push subscription payload", ErrInvalidInput)
	}

	_ = meeting
	return s.repository.AddPushSubscription(
		ctx,
		meetingId,
		strings.TrimSpace(userId),
		strings.TrimSpace(input.DeviceId),
		input.IsStandalone,
		input.NotificationPermissionStatus,
		strings.TrimSpace(input.PushSubscription.Endpoint),
		strings.TrimSpace(input.PushSubscription.Keys.Auth),
		strings.TrimSpace(input.PushSubscription.Keys.P256dh),
	)
}

func (s *MeetingService) RemovePushSubscription(ctx context.Context, meetingId uint32, userId string, deviceId string) error {
	_, err := s.repository.GetMeetingById(ctx, meetingId)
	if err != nil {
		if err == sql.ErrNoRows {
			return ErrNotFound
		}
		return err
	}

	if strings.TrimSpace(userId) == "" {
		return ErrForbidden
	}

	if strings.TrimSpace(deviceId) == "" {
		return fmt.Errorf("%w: device ID is required", ErrInvalidInput)
	}

	return s.repository.RemovePushSubscription(ctx, meetingId, strings.TrimSpace(userId), strings.TrimSpace(deviceId))
}

func (s *MeetingService) GetPushSubscriptionStatus(ctx context.Context, meetingId uint32, userId string, deviceId string) (*PushSubscriptionStatusResponse, error) {
	_, err := s.repository.GetMeetingById(ctx, meetingId)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}

	userId = strings.TrimSpace(userId)
	deviceId = strings.TrimSpace(deviceId)
	if userId == "" {
		return nil, ErrForbidden
	}
	if deviceId == "" {
		return nil, fmt.Errorf("%w: device ID is required", ErrInvalidInput)
	}

	subscription, err := s.repository.GetPushSubscriptionByDevice(ctx, meetingId, userId, deviceId)
	if err != nil {
		return nil, err
	}

	if subscription == nil {
		return &PushSubscriptionStatusResponse{
			MeetingId:                      meetingId,
			UserId:                         userId,
			DeviceId:                       deviceId,
			IsSubscribed:                   false,
			IsStandalone:                   false,
			NotificationPermissionStatus:   "default",
			InstallFlagStatus:              "disabled",
			PushSubscriptionEndpointStatus: "invalid",
			LastVerifiedAt:                 nil,
			LastNudgeAt:                    nil,
		}, nil
	}

	lastVerifiedAt := subscription.LastVerifiedAt
	return &PushSubscriptionStatusResponse{
		MeetingId:                      meetingId,
		UserId:                         userId,
		DeviceId:                       deviceId,
		IsSubscribed:                   subscription.IsActive,
		IsStandalone:                   subscription.IsStandalone,
		NotificationPermissionStatus:   subscription.NotificationPermissionStatus,
		InstallFlagStatus:              "active",
		PushSubscriptionEndpointStatus: normalizeEndpointStatus(subscription.EndpointStatus),
		LastVerifiedAt:                 &lastVerifiedAt,
		LastNudgeAt:                    nil,
	}, nil
}

func (s *MeetingService) GetMyPushSubscriptionStatus(ctx context.Context, meetingId uint32, userId string) ([]*PushSubscriptionStatusResponse, error) {
	_, err := s.repository.GetMeetingById(ctx, meetingId)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}

	userId = strings.TrimSpace(userId)
	if userId == "" {
		return nil, ErrForbidden
	}

	subscriptions, err := s.repository.GetPushSubscriptionsByUser(ctx, meetingId, userId)
	if err != nil {
		return nil, err
	}

	statusList := make([]*PushSubscriptionStatusResponse, 0, len(subscriptions))
	for _, subscription := range subscriptions {
		lastVerifiedAt := subscription.LastVerifiedAt
		statusList = append(statusList, &PushSubscriptionStatusResponse{
			MeetingId:                      meetingId,
			UserId:                         userId,
			DeviceId:                       subscription.DeviceId,
			IsSubscribed:                   subscription.IsActive,
			IsStandalone:                   subscription.IsStandalone,
			NotificationPermissionStatus:   subscription.NotificationPermissionStatus,
			InstallFlagStatus:              "active",
			PushSubscriptionEndpointStatus: normalizeEndpointStatus(subscription.EndpointStatus),
			LastVerifiedAt:                 &lastVerifiedAt,
			LastNudgeAt:                    nil,
		})
	}

	return statusList, nil
}

func (s *MeetingService) SendTestPushToSelf(
	ctx context.Context,
	meetingId uint32,
	userId string,
	input SendTestPushInput,
) (*SendTestPushResponse, error) {
	_, err := s.repository.GetMeetingById(ctx, meetingId)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}

	userId = strings.TrimSpace(userId)
	if userId == "" {
		return nil, ErrForbidden
	}

	if s.pushSender == nil {
		return nil, fmt.Errorf("%w: push sender is not configured", ErrInvalidState)
	}

	subscriptions, err := s.repository.GetPushSubscriptionsByUser(ctx, meetingId, userId)
	if err != nil {
		return nil, err
	}

	activeSubscriptions := make([]*model.NotificationSubscription, 0, len(subscriptions))
	for _, sub := range subscriptions {
		if !sub.IsActive {
			continue
		}
		if strings.ToLower(strings.TrimSpace(sub.NotificationPermissionStatus)) != "granted" {
			continue
		}
		if strings.TrimSpace(normalizeEndpointStatus(sub.EndpointStatus)) != "active" {
			continue
		}
		activeSubscriptions = append(activeSubscriptions, sub)
	}

	if len(activeSubscriptions) == 0 {
		return nil, fmt.Errorf("%w: no active push subscriptions for current user", ErrInvalidState)
	}

	title := strings.TrimSpace(input.Title)
	if title == "" {
		title = "테스트 알림"
	}
	body := strings.TrimSpace(input.Body)
	if body == "" {
		body = "Meet2Meet 수동 테스트 알림입니다."
	}
	url := strings.TrimSpace(input.URL)
	if url == "" {
		url = fmt.Sprintf("/meeting/%d", meetingId)
	}
	tag := strings.TrimSpace(input.Tag)
	if tag == "" {
		tag = fmt.Sprintf("meeting-%d-test", meetingId)
	}

	payload, err := json.Marshal(map[string]any{
		"title": title,
		"body":  body,
		"tag":   tag,
		"data": map[string]any{
			"url":       url,
			"meetingId": meetingId,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("%w: failed to encode push payload", ErrInvalidState)
	}

	resp := &SendTestPushResponse{
		MeetingId:   meetingId,
		SentCount:   0,
		FailCount:   0,
		Results:     make([]SendTestPushResult, 0, len(activeSubscriptions)),
		TriggeredAt: time.Now().UTC(),
	}

	for _, sub := range activeSubscriptions {
		err := s.pushSender.Send(ctx, sub, payload)
		if err != nil {
			if stateErr := updatePushSubscriptionDeliveryState(ctx, s.repository, sub, err); stateErr != nil {
				log.Printf("[push:test] deviceId=%s state update failed: %v", sub.DeviceId, stateErr)
			}
			resp.FailCount++
			resp.Results = append(resp.Results, SendTestPushResult{
				DeviceId: sub.DeviceId,
				Success:  false,
				Error:    err.Error(),
			})
			continue
		}

		resp.SentCount++
		resp.Results = append(resp.Results, SendTestPushResult{
			DeviceId: sub.DeviceId,
			Success:  true,
		})
	}

	return resp, nil
}

func normalizeEndpointStatus(status string) string {
	if status == "" {
		return "invalid"
	}
	return strings.ReplaceAll(status, "_", "-")
}
