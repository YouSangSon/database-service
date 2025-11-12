package grpc_handler

import (
	"context"
	"time"

	"github.com/YouSangSon/database-service/internal/models"
	"github.com/YouSangSon/database-service/internal/service"
	pb "github.com/YouSangSon/database-service/proto/pb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// GRPCHandler는 gRPC 요청을 처리하는 핸들러입니다
type GRPCHandler struct {
	pb.UnimplementedDatabaseServiceServer
	service *service.Service
}

// NewGRPCHandler는 새로운 GRPCHandler 인스턴스를 생성합니다
func NewGRPCHandler(service *service.Service) *GRPCHandler {
	return &GRPCHandler{
		service: service,
	}
}

// Create는 새로운 문서를 생성합니다
func (h *GRPCHandler) Create(ctx context.Context, req *pb.CreateRequest) (*pb.CreateResponse, error) {
	data := req.GetData().AsMap()

	createReq := &models.CreateRequest{
		Collection: req.GetCollection(),
		Data:       data,
	}

	resp, err := h.service.Create(ctx, createReq)
	if err != nil {
		return nil, err
	}

	return &pb.CreateResponse{
		Id:      resp.ID,
		Created: timestamppb.New(resp.Created),
	}, nil
}

// Read는 ID로 문서를 조회합니다
func (h *GRPCHandler) Read(ctx context.Context, req *pb.ReadRequest) (*pb.ReadResponse, error) {
	readReq := &models.ReadRequest{
		Collection: req.GetCollection(),
		ID:         req.GetId(),
	}

	doc, err := h.service.Read(ctx, readReq)
	if err != nil {
		return nil, err
	}

	dataStruct, err := structpb.NewStruct(doc.Data)
	if err != nil {
		return nil, err
	}

	return &pb.ReadResponse{
		Id:        doc.ID,
		Data:      dataStruct,
		CreatedAt: timestamppb.New(doc.CreatedAt),
		UpdatedAt: timestamppb.New(doc.UpdatedAt),
	}, nil
}

// Update는 기존 문서를 업데이트합니다
func (h *GRPCHandler) Update(ctx context.Context, req *pb.UpdateRequest) (*pb.UpdateResponse, error) {
	data := req.GetData().AsMap()

	updateReq := &models.UpdateRequest{
		Collection: req.GetCollection(),
		ID:         req.GetId(),
		Data:       data,
	}

	if err := h.service.Update(ctx, updateReq); err != nil {
		return &pb.UpdateResponse{
			Success: false,
			Message: err.Error(),
		}, err
	}

	return &pb.UpdateResponse{
		Success: true,
		Message: "Document updated successfully",
	}, nil
}

// Delete는 문서를 삭제합니다
func (h *GRPCHandler) Delete(ctx context.Context, req *pb.DeleteRequest) (*pb.DeleteResponse, error) {
	deleteReq := &models.DeleteRequest{
		Collection: req.GetCollection(),
		ID:         req.GetId(),
	}

	if err := h.service.Delete(ctx, deleteReq); err != nil {
		return &pb.DeleteResponse{
			Success: false,
			Message: err.Error(),
		}, err
	}

	return &pb.DeleteResponse{
		Success: true,
		Message: "Document deleted successfully",
	}, nil
}

// List는 컬렉션의 문서 목록을 조회합니다
func (h *GRPCHandler) List(ctx context.Context, req *pb.ListRequest) (*pb.ListResponse, error) {
	var filter map[string]interface{}
	if req.GetFilter() != nil {
		filter = req.GetFilter().AsMap()
	}

	listReq := &models.ListRequest{
		Collection: req.GetCollection(),
		Filter:     filter,
		Limit:      int(req.GetLimit()),
		Skip:       int(req.GetSkip()),
	}

	resp, err := h.service.List(ctx, listReq)
	if err != nil {
		return nil, err
	}

	documents := make([]*pb.Document, 0, len(resp.Documents))
	for _, doc := range resp.Documents {
		dataStruct, err := structpb.NewStruct(doc.Data)
		if err != nil {
			continue
		}

		documents = append(documents, &pb.Document{
			Id:        doc.ID,
			Data:      dataStruct,
			CreatedAt: timestamppb.New(doc.CreatedAt),
			UpdatedAt: timestamppb.New(doc.UpdatedAt),
		})
	}

	return &pb.ListResponse{
		Documents: documents,
		Total:     int32(resp.Total),
	}, nil
}

// HealthCheck는 서비스 상태를 확인합니다
func (h *GRPCHandler) HealthCheck(ctx context.Context, req *pb.HealthCheckRequest) (*pb.HealthCheckResponse, error) {
	if err := h.service.HealthCheck(ctx); err != nil {
		return &pb.HealthCheckResponse{
			Healthy: false,
			Message: err.Error(),
		}, nil
	}

	return &pb.HealthCheckResponse{
		Healthy: true,
		Message: "Service is healthy",
	}, nil
}

// convertToTimestamp는 time.Time을 timestamppb.Timestamp로 변환합니다
func convertToTimestamp(t time.Time) *timestamppb.Timestamp {
	return timestamppb.New(t)
}
