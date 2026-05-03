package handler

import (
	"errors"
	"log"
	"meetBack/internal/service"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type VoteHandler struct {
	service *service.MeetingService
}

func NewVoteHandler(s *service.MeetingService) *VoteHandler {
	return &VoteHandler{service: s}
}

func (h *VoteHandler) GetVotes(c *gin.Context) {
	meetingIdStr := c.Param("meetingId")
	// uint32로 변환
	meetingId, err := strconv.ParseUint(meetingIdStr, 10, 32)
	if err != nil {
		log.Printf("[votes:get] invalid meetingId=%q", meetingIdStr)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid meeting ID"})
		return
	}

	participantCode := c.Query("participantCode")
	hostID := c.GetHeader("X-User-Id")
	log.Printf(
		"[votes:get] request meetingId=%d x_user_id_set=%t participant_code_set=%t",
		meetingId,
		strings.TrimSpace(hostID) != "",
		strings.TrimSpace(participantCode) != "",
	)

	votes, err := h.service.GetVotesByMeetingIdAuthorized(c.Request.Context(), uint32(meetingId), participantCode, hostID)
	if err != nil {
		log.Printf("[votes:get] denied meetingId=%d reason=%v", meetingId, err)
		switch {
		case errors.Is(err, service.ErrForbidden):
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		case errors.Is(err, service.ErrNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}
	log.Printf("[votes:get] success meetingId=%d vote_count=%d", meetingId, len(votes))

	c.JSON(http.StatusOK, gin.H{"votes": votes})
}

// 선택한 슬롯 키 배열. 형식: YYYY-MM-DD-HH:mm (예: "2026-04-20-09:00")
type SubmitVotesRequest struct {
	Slots []string `json:"slots" binding:"required"`
}

func (h *VoteHandler) SubmitVotes(c *gin.Context) {
	meetingIdStr := c.Param("meetingId")
	// uint32로 변환
	meetingId, err := strconv.ParseUint(meetingIdStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid meeting ID"})
		return
	}

	var req SubmitVotesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	participantCode := c.Query("participantCode")
	hostID := c.GetHeader("X-User-Id")
	hostName := c.GetString("userName")
	if strings.TrimSpace(hostName) == "" {
		hostName = strings.TrimSpace(c.GetHeader("X-Participant-Name"))
	}

	err = h.service.SubmitVotesRequest(uint32(meetingId), req.Slots, participantCode, hostID, hostName)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrForbidden):
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		case errors.Is(err, service.ErrNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Votes submitted successfully"})
}

func (h *VoteHandler) DeleteVotes(c *gin.Context) {
	meetingIdStr := c.Param("meetingId")
	// uint32로 변환
	meetingId, err := strconv.ParseUint(meetingIdStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid meeting ID"})
		return
	}

	participantCode := c.Query("participantCode")
	hostID := c.GetHeader("X-User-Id")

	err = h.service.DeleteVotesRequest(uint32(meetingId), participantCode, hostID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrForbidden):
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		case errors.Is(err, service.ErrNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Votes deleted successfully"})
}
