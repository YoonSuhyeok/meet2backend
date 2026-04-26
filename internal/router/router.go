package router

import (
	"meetBack/internal/handler"

	"github.com/gin-gonic/gin"
)

func Register(
	r *gin.Engine,
	healthHandler *handler.HealthHandler,
	meetingHandler *handler.MeetingHandler,
	voteHandler *handler.VoteHandler,
) {
	// 서버 상태 확인
	RegisterHealthRoutes(r, healthHandler)
	// 미팅 관련 라우트
	RegisterMeetingRoutes(r, meetingHandler)
	// 투표 관련 라우트
	RegisterVoteRoutes(r, voteHandler)
}
