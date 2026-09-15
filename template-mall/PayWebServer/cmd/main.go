package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"template-mall/PayWebServer/internal/config"
	"template-mall/PayWebServer/internal/handler"
	"template-mall/PayWebServer/internal/kafka"
	"template-mall/PayWebServer/internal/repository"
	"template-mall/PayWebServer/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	cfg := config.Load()

	db, err := gorm.Open(mysql.Open(cfg.MySQLDSN), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Fatalf("mysql open: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("mysql db: %v", err)
	}
	sqlDB.SetMaxOpenConns(20)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(time.Hour)

	repo := repository.New(db)
	if err := repo.AutoMigrate(); err != nil {
		log.Fatalf("auto migrate: %v", err)
	}

	producer := kafka.NewProducer(cfg.KafkaBrokers, cfg.KafkaTopic, cfg.KafkaTimeout)
	defer producer.Close()

	svc := service.New(repo, producer, cfg.PublicBaseURL)
	r := gin.New()
	r.Use(gin.Recovery(), gin.Logger())
	handler.New(svc, cfg.MallFrontendURL).Register(r)

	srv := &http.Server{Addr: cfg.HTTPAddr, Handler: r}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("PayWebServer listening on %s", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http: %v", err)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
	log.Printf("PayWebServer stopped")
}
