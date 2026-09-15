package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	pb "template-mall/TemplateOrderServer/api/gen/templateorder/v1"
	"template-mall/TemplateOrderServer/internal/config"
	"template-mall/TemplateOrderServer/internal/grpcserver"
	"template-mall/TemplateOrderServer/internal/kafka"
	"template-mall/TemplateOrderServer/internal/payclient"
	"template-mall/TemplateOrderServer/internal/repository"
	"template-mall/TemplateOrderServer/internal/service"
	"template-mall/TemplateOrderServer/internal/storage"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
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

	pay := payclient.New(cfg.PayBaseURL, cfg.PayHTTPTimeout)
	signer, err := storage.NewSigner(storage.Config{
		Provider:   cfg.StorageProvider,
		Endpoint:   cfg.OSSEndpoint,
		AccessKey:  cfg.OSSAccessKey,
		SecretKey:  cfg.OSSSecretKey,
		Bucket:     cfg.OSSBucket,
		MockSecret: cfg.MockSignSecret,
	})
	if err != nil {
		log.Fatalf("storage signer: %v", err)
	}
	svc := service.New(repo, pay, signer, cfg.DownloadURLTTL)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	consumer := kafka.NewConsumer(cfg.KafkaBrokers, cfg.KafkaTopic, cfg.KafkaConsumerGroup, svc)
	defer consumer.Close()
	go consumer.Run(ctx)

	lis, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		log.Fatalf("listen %s: %v", cfg.GRPCAddr, err)
	}
	gs := grpc.NewServer()
	pb.RegisterTemplateOrderServiceServer(gs, grpcserver.New(svc))
	reflection.Register(gs)

	go func() {
		<-ctx.Done()
		log.Printf("shutting down grpc...")
		gs.GracefulStop()
	}()

	log.Printf("TemplateOrderServer gRPC listening on %s", cfg.GRPCAddr)
	if err := gs.Serve(lis); err != nil {
		log.Fatalf("serve: %v", err)
	}
}
