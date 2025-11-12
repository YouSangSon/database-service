package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/YouSangSon/database-service/config"
	"github.com/YouSangSon/database-service/internal/database"
	"github.com/YouSangSon/database-service/internal/database/mongodb"
	"github.com/YouSangSon/database-service/internal/handler"
	"github.com/YouSangSon/database-service/internal/service"
	"github.com/gin-gonic/gin"
)

func main() {
	// 설정 로드
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 데이터베이스 연결
	var db database.Database

	switch cfg.Database.Type {
	case "mongodb":
		db = mongodb.NewMongoDB(&database.Config{
			Type:     cfg.Database.Type,
			Host:     cfg.Database.Host,
			Port:     cfg.Database.Port,
			Username: cfg.Database.Username,
			Password: cfg.Database.Password,
			Database: cfg.Database.Database,
		})
	default:
		log.Fatalf("Unsupported database type: %s", cfg.Database.Type)
	}

	// 데이터베이스 연결
	ctx := context.Background()
	if err := db.Connect(ctx); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Disconnect(ctx)

	log.Printf("Connected to %s database at %s:%d", cfg.Database.Type, cfg.Database.Host, cfg.Database.Port)

	// 서비스 및 핸들러 초기화
	svc := service.NewService(db)
	h := handler.NewHandler(svc)

	// Gin 라우터 설정
	router := gin.Default()

	// 헬스체크 엔드포인트
	router.GET("/health", h.HealthCheck)

	// API v1 그룹
	v1 := router.Group("/api/v1")
	{
		// CRUD 엔드포인트
		v1.POST("/documents", h.Create)
		v1.GET("/documents/:collection/:id", h.Read)
		v1.PUT("/documents/:collection/:id", h.Update)
		v1.DELETE("/documents/:collection/:id", h.Delete)
		v1.GET("/documents/:collection", h.List)
	}

	// 서버 설정
	addr := fmt.Sprintf(":%d", cfg.Server.APIPort)
	srv := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	// 서버 시작
	go func() {
		log.Printf("Starting API server on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}
