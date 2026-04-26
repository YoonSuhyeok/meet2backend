package router

import (
	"meetBack/internal/handler"

	"github.com/gin-gonic/gin"
)

func RegisterVoteRoutes(r gin.IRouter, voteHandler *handler.VoteHandler) {
	// 미팅 투표 결과 조회​
	r.GET("/meetings/:meetingId/votes", voteHandler.GetVotes)
}
