package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"template-mall/TemplateAdminWebServer/internal/auth"
	"template-mall/TemplateAdminWebServer/internal/config"
	"template-mall/TemplateAdminWebServer/internal/grpcclient"
	"template-mall/TemplateAdminWebServer/internal/handler"
	"template-mall/TemplateAdminWebServer/internal/ossstore"
	"template-mall/TemplateAdminWebServer/internal/wpsoauth"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()

	tokens := auth.NewTokenIssuer(cfg.JWTSecret, cfg.JWTTTL)
	oauth := wpsoauth.New(wpsoauth.Config{
		ClientID: cfg.WPSClientID, ClientSecret: cfg.WPSClientSecret,
		RedirectURI: cfg.WPSRedirectURI, AuthURL: cfg.WPSAuthURL,
		TokenURL: cfg.WPSTokenURL, UserInfoURL: cfg.WPSUserInfoURL,
		Scopes: cfg.WPSScopes, HTTPTimeout: 10 * time.Second,
	})

	oss, err := ossstore.New(cfg.OSSProvider, cfg.OSSEndpoint, cfg.OSSAccessKey, cfg.OSSSecretKey, cfg.OSSBucket, cfg.PublicBaseURL, cfg.OSSUploadTTL)
	if err != nil {
		log.Fatalf("oss: %v", err)
	}

	order, err := grpcclient.Dial(cfg.OrderGRPCAddr, cfg.GRPCTimeout)
	if err != nil {
		log.Fatalf("order grpc: %v", err)
	}
	defer order.Close()

	r := gin.New()
	r.Use(gin.Recovery(), gin.Logger())
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type")
		c.Header("Access-Control-Allow-Methods", "GET,POST,PATCH,PUT,OPTIONS")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	})
	handler.New(cfg.AuthMode, tokens, oauth, cfg.FrontendURL, order, oss).Register(r)

	srv := &http.Server{Addr: cfg.HTTPAddr, Handler: r}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		log.Printf("TemplateAdminWebServer on %s auth=%s oss=%s -> order %s", cfg.HTTPAddr, cfg.AuthMode, cfg.OSSProvider, cfg.OrderGRPCAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http: %v", err)
		}
	}()
	<-ctx.Done()
	shCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shCtx)
}
