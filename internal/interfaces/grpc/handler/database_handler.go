package handler

import (
	"context"
	"fmt"
	"time"

	"github.com/YouSangSon/database-service/internal/application/usecase"
	"github.com/YouSangSon/database-service/internal/domain/entity"
	"github.com/YouSangSon/database-service/internal/pkg/logger"
	pb "github.com/YouSangSon/database-service/proto/pb"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// DatabaseHandler는 DatabaseService gRPC 핸들러입니다
type DatabaseHandler struct {
	pb.UnimplementedDatabaseServiceServer
	documentUC *usecase.DocumentUseCase
}

// NewDatabaseHandler는 새로운 DatabaseHandler를 생성합니다
func NewDatabaseHandler(documentUC *usecase.DocumentUseCase) *DatabaseHandler {
	return &DatabaseHandler{
		documentUC: documentUC,
	}
}

// Create는 새로운 문서를 생성합니다
func (h *DatabaseHandler) Create(ctx context.Context, req *pb.CreateRequest) (*pb.CreateResponse, error) {
	logger.Info(ctx, "creating document",
		zap.String("collection", req.Collection),
	)

	// Validate request
	if req.Collection == "" {
		return nil, status.Error(codes.InvalidArgument, "collection is required")
	}
	if req.Data == nil {
		return nil, status.Error(codes.InvalidArgument, "data is required")
	}

	// Convert protobuf Struct to map
	data := req.Data.AsMap()

	// Create document using use case
	doc, err := h.documentUC.Create(ctx, req.Collection, data)
	if err != nil {
		logger.Error(ctx, "failed to create document",
			zap.String("collection", req.Collection),
			zap.Error(err),
		)
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to create document: %v", err))
	}

	logger.Info(ctx, "document created successfully",
		zap.String("collection", req.Collection),
		zap.String("id", doc.ID),
	)

	return &pb.CreateResponse{
		Id:      doc.ID,
		Created: timestamppb.New(doc.CreatedAt),
	}, nil
}

// Read는 ID로 문서를 조회합니다
func (h *DatabaseHandler) Read(ctx context.Context, req *pb.ReadRequest) (*pb.ReadResponse, error) {
	logger.Info(ctx, "reading document",
		zap.String("collection", req.Collection),
		zap.String("id", req.Id),
	)

	// Validate request
	if req.Collection == "" {
		return nil, status.Error(codes.InvalidArgument, "collection is required")
	}
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	// Get document using use case
	doc, err := h.documentUC.GetByID(ctx, req.Collection, req.Id)
	if err != nil {
		if err.Error() == "document not found" {
			return nil, status.Error(codes.NotFound, "document not found")
		}
		logger.Error(ctx, "failed to read document",
			zap.String("collection", req.Collection),
			zap.String("id", req.Id),
			zap.Error(err),
		)
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to read document: %v", err))
	}

	// Convert document data to protobuf Struct
	dataStruct, err := structpb.NewStruct(doc.Data)
	if err != nil {
		logger.Error(ctx, "failed to convert document data",
			zap.Error(err),
		)
		return nil, status.Error(codes.Internal, "failed to convert document data")
	}

	return &pb.ReadResponse{
		Id:        doc.ID,
		Data:      dataStruct,
		CreatedAt: timestamppb.New(doc.CreatedAt),
		UpdatedAt: timestamppb.New(doc.UpdatedAt),
	}, nil
}

// Update는 기존 문서를 업데이트합니다
func (h *DatabaseHandler) Update(ctx context.Context, req *pb.UpdateRequest) (*pb.UpdateResponse, error) {
	logger.Info(ctx, "updating document",
		zap.String("collection", req.Collection),
		zap.String("id", req.Id),
	)

	// Validate request
	if req.Collection == "" {
		return nil, status.Error(codes.InvalidArgument, "collection is required")
	}
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}
	if req.Data == nil {
		return nil, status.Error(codes.InvalidArgument, "data is required")
	}

	// Convert protobuf Struct to map
	data := req.Data.AsMap()

	// Update document using use case
	doc, err := h.documentUC.Update(ctx, req.Collection, req.Id, data)
	if err != nil {
		if err.Error() == "document not found" {
			return nil, status.Error(codes.NotFound, "document not found")
		}
		logger.Error(ctx, "failed to update document",
			zap.String("collection", req.Collection),
			zap.String("id", req.Id),
			zap.Error(err),
		)
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to update document: %v", err))
	}

	logger.Info(ctx, "document updated successfully",
		zap.String("collection", req.Collection),
		zap.String("id", doc.ID),
	)

	return &pb.UpdateResponse{
		Success: true,
		Message: "document updated successfully",
	}, nil
}

// Delete는 문서를 삭제합니다
func (h *DatabaseHandler) Delete(ctx context.Context, req *pb.DeleteRequest) (*pb.DeleteResponse, error) {
	logger.Info(ctx, "deleting document",
		zap.String("collection", req.Collection),
		zap.String("id", req.Id),
	)

	// Validate request
	if req.Collection == "" {
		return nil, status.Error(codes.InvalidArgument, "collection is required")
	}
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	// Delete document using use case
	err := h.documentUC.Delete(ctx, req.Collection, req.Id)
	if err != nil {
		if err.Error() == "document not found" {
			return nil, status.Error(codes.NotFound, "document not found")
		}
		logger.Error(ctx, "failed to delete document",
			zap.String("collection", req.Collection),
			zap.String("id", req.Id),
			zap.Error(err),
		)
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to delete document: %v", err))
	}

	logger.Info(ctx, "document deleted successfully",
		zap.String("collection", req.Collection),
		zap.String("id", req.Id),
	)

	return &pb.DeleteResponse{
		Success: true,
		Message: "document deleted successfully",
	}, nil
}

// List는 문서 목록을 조회합니다
func (h *DatabaseHandler) List(ctx context.Context, req *pb.ListRequest) (*pb.ListResponse, error) {
	logger.Info(ctx, "listing documents",
		zap.String("collection", req.Collection),
		zap.Int32("limit", req.Limit),
		zap.Int32("skip", req.Skip),
	)

	// Validate request
	if req.Collection == "" {
		return nil, status.Error(codes.InvalidArgument, "collection is required")
	}

	// Default limit
	limit := req.Limit
	if limit == 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	// Convert protobuf Struct to map for filter
	var filter map[string]interface{}
	if req.Filter != nil {
		filter = req.Filter.AsMap()
	}

	// List documents using use case
	docs, err := h.documentUC.List(ctx, req.Collection, int(limit), int(req.Skip))
	if err != nil {
		logger.Error(ctx, "failed to list documents",
			zap.String("collection", req.Collection),
			zap.Error(err),
		)
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to list documents: %v", err))
	}

	// Convert documents to protobuf
	pbDocs := make([]*pb.Document, len(docs))
	for i, doc := range docs {
		dataStruct, err := structpb.NewStruct(doc.Data)
		if err != nil {
			logger.Error(ctx, "failed to convert document data",
				zap.String("id", doc.ID),
				zap.Error(err),
			)
			continue
		}

		pbDocs[i] = &pb.Document{
			Id:        doc.ID,
			Data:      dataStruct,
			CreatedAt: timestamppb.New(doc.CreatedAt),
			UpdatedAt: timestamppb.New(doc.UpdatedAt),
		}
	}

	return &pb.ListResponse{
		Documents: pbDocs,
		Total:     int32(len(docs)),
	}, nil
}

// HealthCheck는 서비스 상태를 확인합니다
func (h *DatabaseHandler) HealthCheck(ctx context.Context, req *pb.HealthCheckRequest) (*pb.HealthCheckResponse, error) {
	// Simple health check - can be enhanced with dependency checks
	return &pb.HealthCheckResponse{
		Healthy: true,
		Message: "service is healthy",
	}, nil
}

// convertToEntity는 protobuf Document를 entity.Document로 변환합니다
func convertToEntity(pbDoc *pb.Document) *entity.Document {
	return &entity.Document{
		ID:        pbDoc.Id,
		Data:      pbDoc.Data.AsMap(),
		CreatedAt: pbDoc.CreatedAt.AsTime(),
		UpdatedAt: pbDoc.UpdatedAt.AsTime(),
	}
}

// convertFromEntity는 entity.Document를 protobuf Document로 변환합니다
func convertFromEntity(doc *entity.Document) (*pb.Document, error) {
	dataStruct, err := structpb.NewStruct(doc.Data)
	if err != nil {
		return nil, err
	}

	return &pb.Document{
		Id:        doc.ID,
		Data:      dataStruct,
		CreatedAt: timestamppb.New(doc.CreatedAt),
		UpdatedAt: timestamppb.New(doc.UpdatedAt),
	}, nil
}
