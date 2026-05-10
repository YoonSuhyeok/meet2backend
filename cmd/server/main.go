package main

import (
	"context"
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

	appCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	healthHandler := handler.NewHealthHandler(db)

	meetingRepository := repository.NewMeetingRepository(db)
	var pushSender service.PushSender
	if cfg.VapidPublicKey != "" && cfg.VapidPrivateKey != "" {
		pushSender = service.NewWebPushSender(
			cfg.VapidSubject,
			cfg.VapidPublicKey,
			cfg.VapidPrivateKey,
		)
	} else {
		log.Printf("[push] disabled: VAPID_PUBLIC_KEY / VAPID_PRIVATE_KEY is not configured")
	}
	meetingService := service.NewMeetingService(meetingRepository, pushSender)
	if pushSender != nil {
		go service.NewAttendanceNudgeWorker(meetingRepository, pushSender).Run(appCtx)
	}
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
