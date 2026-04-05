package grpc

import (
	"context"
	"log/slog"

	pb "ingestion/api/gen/ingestion"
	"ingestion/internal/cache"
	"ingestion/internal/db"
	"ingestion/internal/kafka"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type IngestionService struct {
	pb.UnimplementedIngestionServiceServer
	store *db.PaymentStore
	dedupe *cache.DedupeCache
	publisher *kafka.PaymentEventPublisher
	logger *slog.Logger
}

func NewIngestionService(store *db.PaymentStore, dedupe *cache.DedupeCache, publisher *kafka.PaymentEventPublisher, logger *slog.Logger) *IngestionService {
	return &IngestionService{
		store: store,
		dedupe: dedupe,
		publisher: publisher,
		logger: logger,
	}
}

func (s *IngestionService) IngestPayment(ctx context.Context, req *pb.IngestPaymentRequest) (*pb.IngestPaymentResponse, error) {
	if req.IdempotencyKey == "" || req.TenantId == "" || req.Currency == "" || req.Source == "" || req.Destination == "" {
		return nil, status.Error(codes.InvalidArgument, "idempotency_key, tenant_id, currency, source, and destination are required")
	}

	if req.Amount <= 0 {
		return nil, status.Error(codes.InvalidArgument, "amount must be greater than 0")
	}

	log := s.logger.With("tenant_id", req.TenantId, "idempotency_key", req.IdempotencyKey)

	dup, err := s.dedupe.IsDuplicate(ctx, req.TenantId, req.IdempotencyKey)
	if err != nil {
		log.Warn("failed to check deduplication cache", "error", err)
	}

	if dup {
		log.Info("duplicate detecte via cache")
		return &pb.IngestPaymentResponse{
			Result: &pb.IngestionResult{
				Status: pb.PaymentStatus_PAYMENT_STATUS_DUPLICATE,
				Reason: "duplicate idempotency key",
			},
		}, nil
	}

	evt := &db.PaymentEvent{
		IdempotencyKey: req.IdempotencyKey,
		TenantID: req.TenantId,
		Amount: req.Amount,
		Currency: req.Currency,
		Source: req.Source,
		Destination: req.Destination,
		Metadata: req.Metadata,
	}

	inserted, isDup, err := s.store.InsertPaymentEvent(ctx, evt)
	if err != nil {
		log.Error("failed to insert payment event", "error", err)
		return nil, status.Error(codes.Internal, "failed to process payment")
	}

	if isDup {
	log.Info("duplicate detected via database")
		return &pb.IngestPaymentResponse{
			Result: &pb.IngestionResult{
				Status: pb.PaymentStatus_PAYMENT_STATUS_DUPLICATE,
				Reason: "duplicate idempotency key",
			},
		}, nil
	}

	if err := s.dedupe.MarkProcessed(ctx, req.TenantId, req.IdempotencyKey); err != nil {
		log.Warn("failed to mark idempotency key as processed in cache", "error", err)
	}

	if err := s.publisher.PublishPaymentReceived(ctx, evt); err != nil {
		log.Warn("failed to publish payment event to Kafka", "error", err)
	}

	log.Info("payment ingested", "event_id", inserted.ID)

	return &pb.IngestPaymentResponse{
		Result: &pb.IngestionResult{
			EventId: inserted.ID,
			Status: pb.PaymentStatus_PAYMENT_STATUS_RECEIVED,
		},
	}, nil
}	

func toProtoEvent(e *db.PaymentEvent) *pb.PaymentEvent {
    return &pb.PaymentEvent{
        Id:             e.ID,
        IdempotencyKey: e.IdempotencyKey,
        TenantId:       e.TenantID,
        Amount:         e.Amount,
        Currency:       e.Currency,
        Source:         e.Source,
        Destination:    e.Destination,
        Metadata:       e.Metadata,
        ReceivedAt:     timestamppb.New(e.ReceivedAt),
        Status:         toProtoStatus(e.Status),
    }
}

func toProtoStatus(s db.PaymentStatus) pb.PaymentStatus {
    switch s {
    case db.PaymentStatusReceived:
        return pb.PaymentStatus_PAYMENT_STATUS_RECEIVED
    case db.PaymentStatusRejected:
        return pb.PaymentStatus_PAYMENT_STATUS_REJECTED
    default:
        return pb.PaymentStatus_PAYMENT_STATUS_UNSPECIFIED
    }
}