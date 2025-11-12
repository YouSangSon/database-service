package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/YouSangSon/database-service/config"
	"github.com/YouSangSon/database-service/internal/database"
	"github.com/YouSangSon/database-service/internal/database/mongodb"
	"github.com/YouSangSon/database-service/internal/grpc_handler"
	"github.com/YouSangSon/database-service/internal/service"
	pb "github.com/YouSangSon/database-service/proto/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
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
	h := grpc_handler.NewGRPCHandler(svc)

	// gRPC 서버 생성
	addr := fmt.Sprintf(":%d", cfg.Server.GRPCPort)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterDatabaseServiceServer(grpcServer, h)

	// gRPC reflection 활성화 (gRPC 클라이언트 도구 사용을 위해)
	reflection.Register(grpcServer)

	// 서버 시작
	go func() {
		log.Printf("Starting gRPC server on %s", addr)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down gRPC server...")
	grpcServer.GracefulStop()
	log.Println("gRPC server exited")
}
