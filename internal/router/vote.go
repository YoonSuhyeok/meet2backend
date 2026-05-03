package router

import (
	"meetBack/internal/handler"
	"meetBack/internal/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterVoteRoutes(r gin.IRouter, voteHandler *handler.VoteHandler) {
	// 투표 조회는 participantCode 또는 hostId 기반으로 핸들러/서비스에서 권한 확인
	r.GET("/meetings/:meetingId/votes", voteHandler.GetVotes)
	// 공유 링크 참여자는 비로그인으로도 제출 가능
	r.PUT("/meetings/:meetingId/votes", voteHandler.SubmitVotes)

	// 인증 필요 라우트
	auth := r.Group("/")
	auth.Use(middleware.Auth())
	// 참여자가 자신의 투표를 삭제
	auth.DELETE("/meetings/:meetingId/votes", voteHandler.DeleteVotes)
}
