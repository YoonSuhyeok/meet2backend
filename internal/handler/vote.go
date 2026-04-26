package handler

import (
	"errors"
	"meetBack/internal/service"
	"net/http"
	"strconv"

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
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid meeting ID"})
		return
	}

	participantCode := c.Query("participantCode")
	hostID := c.GetHeader("X-User-Id")

	votes, err := h.service.GetVotesByMeetingIdAuthorized(c.Request.Context(), uint32(meetingId), participantCode, hostID)
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

	c.JSON(http.StatusOK, gin.H{"votes": votes})
}
