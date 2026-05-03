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
	auth.GET("/meetings/:meetingId", h.GetMeetingById)
	auth.PATCH("/meetings/:meetingId", h.UpdateMeeting)
	auth.DELETE("/meetings/:meetingId", h.DeleteMeeting)
	auth.GET("/meetings/:meetingId/join-requests", h.ListJoinRequests)
	auth.POST("/meetings/:meetingId/join-requests/:requestId/approve", h.ApproveJoinRequest)
	auth.POST("/meetings/:meetingId/join-requests/:requestId/reject", h.RejectJoinRequest)
	// PushNotification 구독 등록​
	auth.POST("/meetings/:meetingId/push-subscriptions", h.AddPushSubscription)
	// PushNotification 구독 상태 조회 (현재 기기)
	auth.GET("/meetings/:meetingId/push-subscriptions/status", h.GetPushSubscriptionStatus)
	// PushNotification 구독 해지
	auth.DELETE("/meetings/:meetingId/push-subscriptions", h.RemovePushSubscription)
	// 내 구독 상태 확인 (기기별)
	auth.GET("/meetings/:meetingId/push-subscriptions", h.GetMyPushSubscriptionStatus)
}
