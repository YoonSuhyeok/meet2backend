package handler

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"meetBack/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/uptrace/bun"
)

type MeetingHandler struct {
	service *service.MeetingService
}

func NewMeetingHandler(s *service.MeetingService) *MeetingHandler {
	return &MeetingHandler{service: s}
}

type HealthHandler struct {
	db *bun.DB
}

func NewHealthHandler(db *bun.DB) *HealthHandler {
	return &HealthHandler{db: db}
}

func (h *HealthHandler) Hello(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message":  "meetBack server is running",
		"database": "postgresql",
	})
}

func (h *HealthHandler) Health(c *gin.Context) {
	if err := h.db.PingContext(c.Request.Context()); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "unhealthy",
			"error":  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
	})
}

type createMeetingRequest struct {
	Title       string   `json:"title" binding:"required,min=1,max=50"`
	Description string   `json:"description" binding:"omitempty,max=200"`
	Location    string   `json:"location" binding:"omitempty,max=100"`
	StartTime   string   `json:"startTime" binding:"required,start_time"`
	EndTime     string   `json:"endTime" binding:"required,end_time"`
	Dates       []string `json:"dates" binding:"required,min=1,max=10"`
	JoinPolicy  string   `json:"joinPolicy" binding:"omitempty,oneof=auto approval"`
}

type meetingRequest struct {
	Cursor string `json:"cursor" binding:"required"`
	Limit  string `json:"limit" binding:"default=20"`
}

func (h *MeetingHandler) CreateMeeting(c *gin.Context) {
	var req createMeetingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hostID := c.GetString("userId")
	hostName := c.GetString("userName")
	if hostID == "" || hostName == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing user headers"})
		return
	}

	in := service.CreateMeetingInput{
		Title:       req.Title,
		Description: req.Description,
		Location:    req.Location,
		StartTime:   req.StartTime,
		EndTime:     req.EndTime,
		Dates:       req.Dates,
		JoinPolicy:  req.JoinPolicy,
		HostId:      hostID,
		HostName:    hostName,
	}

	meeting, err := h.service.CreateMeeting(c.Request.Context(), in)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, meeting)
}

func (h *MeetingHandler) GetMeetings(c *gin.Context) {
	var req meetingRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	limit, err := strconv.ParseUint(req.Limit, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid limit"})
		return
	}

	meetings, nextCursor, err := h.service.GetMeetings(c.Request.Context(), req.Cursor, uint32(limit))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"meetings":   meetings,
		"nextCursor": nextCursor,
	})
}

type updateMeetingRequest struct {
	Title       string   `json:"title" binding:"omitempty, min=1,max=50"`
	Description string   `json:"description" binding:"omitempty,max=200"`
	Location    string   `json:"location" binding:"omitempty,max=100"`
	StartTime   string   `json:"startTime" binding:"omitempty,start_time"`
	EndTime     string   `json:"endTime" binding:"omitempty,end_time"`
	Dates       []string `json:"dates" binding:"omitempty,min=1,max=10"`
}

/*
미팅 주최자만 수정 가능. 변경할 필드만 전송 (partial update).
*/
func (h *MeetingHandler) UpdateMeeting(c *gin.Context) {
	meetingId := c.Param("meetingId")
	// meetingId가 uint로 변환 가능한지 검증
	if _, err := strconv.ParseUint(meetingId, 10, 32); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid meetingId"})
		return
	}

	var req updateMeetingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// req가 모두 비어있으면 400
	if req.Title == "" && req.Description == "" && req.Location == "" && req.StartTime == "" && req.EndTime == "" && len(req.Dates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no fields to update"})
		return
	}

	in := service.UpdateMeetingInput{
		Title:       req.Title,
		Description: req.Description,
		Location:    req.Location,
		StartTime:   req.StartTime,
		EndTime:     req.EndTime,
		Dates:       req.Dates,
		HostId:      c.GetString("userId"),
	}

	meetingIdUint, err := strconv.ParseUint(meetingId, 10, 32)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	_, err = h.service.UpdateMeeting(c.Request.Context(), uint32(meetingIdUint), in)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "meeting updated"})
}

func (h *MeetingHandler) DeleteMeeting(c *gin.Context) {
	meetingId := c.Param("meetingId")
	// meetingId가 uint로 변환 가능한지 검증
	if _, err := strconv.ParseUint(meetingId, 10, 32); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid meetingId"})
		return
	}

	hostID := c.GetHeader("X-User-Id")
	if hostID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing user header"})
		return
	}

	meetingIdUint, err := strconv.ParseUint(meetingId, 10, 32)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	err = h.service.DeleteMeeting(c.Request.Context(), uint32(meetingIdUint), hostID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "meeting deleted"})
}

type timeRange struct {
	StartTime string `json:"startTime"`
	EndTime   string `json:"endTime"`
}

type meetingWithVotesResponse struct {
	CreatedAt        time.Time   `json:"createdAt"`
	Dates            []time.Time `json:"dates"`
	Description      string      `json:"description"`
	HostId           string      `json:"hostId"`
	HostName         string      `json:"hostName"`
	Id               uint32      `json:"id"`
	InviteCode       string      `json:"inviteCode"`
	InvitePolicy     string      `json:"invitePolicy"`
	Location         string      `json:"location"`
	ParticipantCount int         `json:"participantCount"`
	ShortId          string      `json:"shortId"`
	TimeRange        timeRange   `json:"timeRange"`
	Title            string      `json:"title"`
	UpdatedAt        time.Time   `json:"updatedAt"`
}

func (h *MeetingHandler) GetMeetingByInviteCode(c *gin.Context) {
	// inviteCode -> ^[A-Z0-9]{3}-[A-Z0-9]{4}$
	inviteCode := c.Param("inviteCode")
	if matched := validateInviteCode(inviteCode); !matched {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid invite code format"})
		return
	}

	meeting, err := h.service.GetMeetingByInviteCode(c.Request.Context(), inviteCode)
	if err != nil {
		writeServiceError(c, err)
		return
	}

	votes, err := h.service.GetVotesByMeetingId(c.Request.Context(), meeting.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	response := meetingWithVotesResponse{
		CreatedAt:        meeting.CreatedAt,
		Dates:            meeting.Dates,
		Description:      meeting.Description,
		HostId:           meeting.HostId,
		HostName:         meeting.HostName,
		Id:               meeting.ID,
		InviteCode:       meeting.InviteCode,
		InvitePolicy:     meeting.InvitePolicy,
		Location:         meeting.Location,
		ParticipantCount: len(votes),
		ShortId:          meeting.ShortId,
		TimeRange: timeRange{
			StartTime: meeting.StartTime,
			EndTime:   meeting.EndTime,
		},
		Title:     meeting.Title,
		UpdatedAt: meeting.UpdatedAt,
	}

	c.JSON(http.StatusOK, response)
}

type createJoinRequestRequest struct {
	RequesterId   string `json:"requesterId" binding:"required,min=1,max=64"`
	RequesterName string `json:"requesterName" binding:"required,min=1,max=50"`
}

func (h *MeetingHandler) CreateJoinRequest(c *gin.Context) {
	inviteCode := c.Param("inviteCode")
	if matched := validateInviteCode(inviteCode); !matched {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid invite code format"})
		return
	}

	var req createJoinRequestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.service.CreateJoinRequestByInviteCode(c.Request.Context(), service.JoinRequestCreateInput{
		InviteCode:    inviteCode,
		RequesterId:   req.RequesterId,
		RequesterName: req.RequesterName,
	})
	if err != nil {
		writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *MeetingHandler) ListJoinRequests(c *gin.Context) {
	meetingId, ok := parseUint32Param(c, "meetingId")
	if !ok {
		return
	}

	hostID := c.GetString("userId")
	requests, err := h.service.GetJoinRequestsForHost(c.Request.Context(), meetingId, hostID)
	if err != nil {
		writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"requests": requests})
}

func (h *MeetingHandler) ApproveJoinRequest(c *gin.Context) {
	meetingId, ok := parseUint32Param(c, "meetingId")
	if !ok {
		return
	}
	requestId, ok := parseUint32Param(c, "requestId")
	if !ok {
		return
	}

	hostID := c.GetString("userId")
	result, err := h.service.ApproveJoinRequest(c.Request.Context(), meetingId, requestId, hostID)
	if err != nil {
		writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"request":         result.Request,
		"participantCode": result.ParticipantCode,
	})
}

func (h *MeetingHandler) RejectJoinRequest(c *gin.Context) {
	meetingId, ok := parseUint32Param(c, "meetingId")
	if !ok {
		return
	}
	requestId, ok := parseUint32Param(c, "requestId")
	if !ok {
		return
	}

	hostID := c.GetString("userId")
	request, err := h.service.RejectJoinRequest(c.Request.Context(), meetingId, requestId, hostID)
	if err != nil {
		writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"request": request})
}

type verifyParticipantCodeRequest struct {
	ParticipantCode string `json:"participantCode" binding:"required,min=8,max=24"`
}

func (h *MeetingHandler) VerifyParticipantCode(c *gin.Context) {
	meetingId, ok := parseUint32Param(c, "meetingId")
	if !ok {
		return
	}

	var req verifyParticipantCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	participant, valid, err := h.service.VerifyParticipantCode(c.Request.Context(), meetingId, req.ParticipantCode)
	if err != nil {
		writeServiceError(c, err)
		return
	}

	response := gin.H{
		"valid":     valid,
		"meetingId": meetingId,
	}

	if valid {
		response["requesterId"] = participant.RequesterId
		response["requesterName"] = participant.RequesterName
	}

	c.JSON(http.StatusOK, response)
}

// [A-Z0-9]{3}-[A-Z0-9]{4}$ 검증 코드
func validateInviteCode(code string) bool {
	if len(code) != 8 {
		return false
	}
	for i, r := range code {
		if i == 3 {
			if r != '-' {
				return false
			}
		} else {
			if !(r >= 'A' && r <= 'Z' || r >= '0' && r <= '9') {
				return false
			}
		}
	}
	return true
}

func (h *MeetingHandler) GetMeetingByShortId(c *gin.Context) {

}

func parseUint32Param(c *gin.Context, name string) (uint32, bool) {
	v, err := strconv.ParseUint(c.Param(name), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid " + name})
		return 0, false
	}

	return uint32(v), true
}

func writeServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrForbidden):
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrInvalidState):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}
