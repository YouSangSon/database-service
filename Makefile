.PHONY: proto build run-api run-grpc docker-build docker-up docker-down test clean

# Proto 파일 컴파일
proto:
	@echo "Generating gRPC code from proto files..."
	mkdir -p proto/pb
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		proto/database.proto

# Go 빌드
build:
	@echo "Building application..."
	go build -o bin/api cmd/api/main.go
	go build -o bin/grpc cmd/grpc/main.go

# API 서버 실행
run-api:
	@echo "Starting API server..."
	go run cmd/api/main.go

# gRPC 서버 실행
run-grpc:
	@echo "Starting gRPC server..."
	go run cmd/grpc/main.go

# Docker 빌드
docker-build:
	@echo "Building Docker images..."
	docker-compose build

# Docker 실행
docker-up:
	@echo "Starting services with Docker Compose..."
	docker-compose up -d

# Docker 중지
docker-down:
	@echo "Stopping services..."
	docker-compose down

# 테스트 실행
test:
	@echo "Running tests..."
	go test -v ./...

# 의존성 다운로드
deps:
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy

# 클린업
clean:
	@echo "Cleaning up..."
	rm -rf bin/
	rm -rf proto/pb/

# 모든 작업 실행
all: proto deps build
