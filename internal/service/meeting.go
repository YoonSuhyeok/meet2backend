package service

import (
	"context"
	"crypto/rand"
	"database/sql"
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

var (
	ErrForbidden    = errors.New("forbidden")
	ErrNotFound     = errors.New("not found")
	ErrInvalidState = errors.New("invalid state")
	ErrInvalidInput = errors.New("invalid input")
)

func NewMeetingService(repository *repository.MeetingRepository) *MeetingService {
	return &MeetingService{repository: repository}
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

func (s *MeetingService) GetMeetings(ctx context.Context, cursor string, limit uint32) ([]*model.Meeting, string, error) {
	return s.repository.GetMeetings(ctx, cursor, limit)
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

	participantCode = strings.TrimSpace(participantCode)
	if participantCode == "" {
		log.Printf(
			"[votes:auth] meetingId=%d access=denied reason=missing_participant_code host_id_set=%t",
			meetingId,
			strings.TrimSpace(hostId) != "",
		)
		return nil, ErrForbidden
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

func (s *MeetingService) SubmitVotesRequest(meetingId uint32, selectedSlots []string, participantCode string, hostId string) error {
	meeting, err := s.repository.GetMeetingById(context.Background(), meetingId)
	if err != nil {
		if err == sql.ErrNoRows {
			return ErrNotFound
		}
		return err
	}

	if hostId != "" && hostId == meeting.HostId {
		return s.repository.SubmitVotes(context.Background(), meetingId, selectedSlots, participantCode)
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

	return s.repository.SubmitVotes(context.Background(), meetingId, selectedSlots, participantCode)
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
