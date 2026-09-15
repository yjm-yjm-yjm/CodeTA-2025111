package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"template-mall/TemplateWebServer/internal/auth"
	"template-mall/TemplateWebServer/internal/config"
	"template-mall/TemplateWebServer/internal/grpcclient"
	"template-mall/TemplateWebServer/internal/handler"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()

	store, err := auth.NewStore(cfg.AuthUsersFile)
	if err != nil {
		log.Fatalf("auth store: %v", err)
	}
	tokens := auth.NewTokenIssuer(cfg.JWTSecret, cfg.JWTTTL)

	order, err := grpcclient.Dial(cfg.OrderGRPCAddr, cfg.GRPCTimeout)
	if err != nil {
		log.Fatalf("order grpc: %v", err)
	}
	defer order.Close()

	r := gin.New()
	r.Use(gin.Recovery(), gin.Logger())
	// 简单 CORS，便于后续前端本地联调
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type")
		c.Header("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	})
	handler.New(store, tokens, order).Register(r)

	srv := &http.Server{Addr: cfg.HTTPAddr, Handler: r}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("TemplateWebServer listening on %s -> order %s", cfg.HTTPAddr, cfg.OrderGRPCAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http: %v", err)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
	log.Printf("TemplateWebServer stopped")
}
