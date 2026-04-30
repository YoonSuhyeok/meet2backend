package router

import (
	"meetBack/internal/handler"
	"meetBack/internal/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterVoteRoutes(r gin.IRouter, voteHandler *handler.VoteHandler) {
	// 투표 조회는 participantCode 또는 hostId 기반으로 핸들러/서비스에서 권한 확인
	r.GET("/meetings/:meetingId/votes", voteHandler.GetVotes)

	// 인증 필요 라우트
	auth := r.Group("/")
	auth.Use(middleware.Auth())
	// 시간 투표 제출 (생성/수정)
	auth.PUT("/meetings/:meetingId/votes", voteHandler.SubmitVotes)
	// 참여자가 자신의 투표를 삭제
	auth.DELETE("/meetings/:meetingId/votes", voteHandler.DeleteVotes)
}
