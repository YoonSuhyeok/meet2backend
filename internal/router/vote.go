package router

import (
	"meetBack/internal/handler"
	"meetBack/internal/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterVoteRoutes(r gin.IRouter, voteHandler *handler.VoteHandler) {
	// 인증 필요 라우트
	auth := r.Group("/")
	auth.Use(middleware.Auth())
	// 시간 투표 제출 (생성/수정)
	auth.PUT("/meetings/:meetingId/votes", voteHandler.SubmitVotes)
	// 미팅 투표 결과 조회​
	auth.GET("/meetings/:meetingId/votes", voteHandler.GetVotes)
	// 참여자가 자신의 투표를 삭제
	auth.DELETE("/meetings/:meetingId/votes", voteHandler.DeleteVotes)
}
