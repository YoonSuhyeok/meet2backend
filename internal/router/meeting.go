package router

import (
	"meetBack/internal/handler"
	"meetBack/internal/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterMeetingRoutes(r gin.IRouter, h *handler.MeetingHandler) {
	// 공개 조회/참가 라우트
	r.GET("/meetings/code/:inviteCode", h.GetMeetingByInviteCode)
	r.GET("/meetings/s/:shortId", h.GetMeetingByShortId)
	r.POST("/meetings/code/:inviteCode/join-requests", h.CreateJoinRequest)
	r.POST("/meetings/:meetingId/participant-codes/verify", h.VerifyParticipantCode)

	// 인증 필요 라우트
	auth := r.Group("/")
	auth.Use(middleware.Auth())
	auth.POST("/meetings", h.CreateMeeting)
	auth.GET("/meetings", h.GetMeetings)
	auth.PATCH("/meetings/:meetingId", h.UpdateMeeting)
	auth.DELETE("/meetings/:meetingId", h.DeleteMeeting)
	auth.GET("/meetings/:meetingId/join-requests", h.ListJoinRequests)
	auth.POST("/meetings/:meetingId/join-requests/:requestId/approve", h.ApproveJoinRequest)
	auth.POST("/meetings/:meetingId/join-requests/:requestId/reject", h.RejectJoinRequest)
}
