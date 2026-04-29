package main

import (
	"log"

	"meetBack/internal/config"
	"meetBack/internal/handler"
	"meetBack/internal/repository"
	"meetBack/internal/router"
	"meetBack/internal/service"
	"meetBack/internal/validator"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()

	db, err := config.OpenDB(cfg)
	if err != nil {
		log.Fatalf("connect postgres: %v", err)
	}
	defer db.Close()

	healthHandler := handler.NewHealthHandler(db)

	meetingRepository := repository.NewMeetingRepository(db)
	meetingService := service.NewMeetingService(meetingRepository)
	meetingHandler := handler.NewMeetingHandler(meetingService)
	voteHandler := handler.NewVoteHandler(meetingService)

	r := gin.Default()

	validator.RegisterValidators()

	router.Register(
		r,
		healthHandler,
		meetingHandler,
		voteHandler)

	if err := r.Run(":" + cfg.AppPort); err != nil {
		log.Fatalf("run gin server: %v", err)
	}
}
