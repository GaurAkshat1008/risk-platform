package grpc

import (
	"context"
	"errors"
	"log/slog"

	pb "notification/api/gen/notification"
	"notification/internal/cache"
	"notification/internal/db"
	"notification/internal/telemetry"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// NotificationService implements the gRPC NotificationService server.
type NotificationService struct {
	pb.UnimplementedNotificationServiceServer
	store       *db.NotificationStore
	rateLimiter *cache.RateLimiter
	metrics     *telemetry.Metrics
	logger      *slog.Logger
}

func NewNotificationService(
	store *db.NotificationStore,
	rateLimiter *cache.RateLimiter,
	metrics *telemetry.Metrics,
	logger *slog.Logger,
) *NotificationService {
	return &NotificationService{
		store:       store,
		rateLimiter: rateLimiter,
		metrics:     metrics,
		logger:      logger,
	}
}

// ── SendNotification ──────────────────────────────────────────────────────────

func (s *NotificationService) SendNotification(
	ctx context.Context,
	req *pb.SendNotificationRequest,
) (*pb.SendNotificationResponse, error) {
	if req.TenantId == "" {
		return nil, status.Error(codes.InvalidArgument, "tenant_id is required")
	}
	if req.Type == "" {
		return nil, status.Error(codes.InvalidArgument, "type is required")
	}
	if req.Recipient == "" {
		return nil, status.Error(codes.InvalidArgument, "recipient is required")
	}

	channelStr := channelToString(req.Channel)

	// Rate limit per tenant/channel.
	allowed, err := s.rateLimiter.Allow(ctx, req.TenantId, channelStr)
	if err != nil {
		s.logger.Warn("rate limiter error, allowing by default", "error", err)
		allowed = true
	}
	if !allowed {
		s.metrics.RateLimitHitsTotal.Add(ctx, 1,
			metric.WithAttributes(attribute.String("tenant_id", req.TenantId)))
		return nil, status.Errorf(codes.ResourceExhausted,
			"rate limit exceeded for tenant %s on channel %s", req.TenantId, channelStr)
	}

	n := &db.Notification{
		TenantID:  req.TenantId,
		Type:      req.Type,
		Recipient: req.Recipient,
		Channel:   channelStr,
		Payload:   req.Payload,
	}
	id, err := s.store.CreateNotification(ctx, n)
	if err != nil {
		s.logger.Error("CreateNotification store error", "error", err)
		return nil, status.Errorf(codes.Internal, "create notification: %v", err)
	}

	s.metrics.NotificationsSentTotal.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("tenant_id", req.TenantId),
			attribute.String("channel", channelStr),
		))
	s.logger.Info("notification created",
		"notification_id", id, "tenant_id", req.TenantId, "type", req.Type)

	return &pb.SendNotificationResponse{
		NotificationId: id,
		Status:         pb.NotificationStatus_NOTIFICATION_STATUS_PENDING,
	}, nil
}

// ── GetDeliveryStatus ─────────────────────────────────────────────────────────

func (s *NotificationService) GetDeliveryStatus(
	ctx context.Context,
	req *pb.GetDeliveryStatusRequest,
) (*pb.GetDeliveryStatusResponse, error) {
	if req.NotificationId == "" {
		return nil, status.Error(codes.InvalidArgument, "notification_id is required")
	}
	if req.TenantId == "" {
		return nil, status.Error(codes.InvalidArgument, "tenant_id is required")
	}

	n, err := s.store.GetNotification(ctx, req.NotificationId, req.TenantId)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return nil, status.Errorf(codes.NotFound, "notification %s not found", req.NotificationId)
		}
		s.logger.Error("GetNotification store error", "error", err)
		return nil, status.Errorf(codes.Internal, "get notification: %v", err)
	}

	return &pb.GetDeliveryStatusResponse{Notification: notificationToProto(n)}, nil
}

// ── UpdateNotificationPreferences ────────────────────────────────────────────

func (s *NotificationService) UpdateNotificationPreferences(
	ctx context.Context,
	req *pb.UpdateNotificationPreferencesRequest,
) (*pb.UpdateNotificationPreferencesResponse, error) {
	if req.TenantId == "" {
		return nil, status.Error(codes.InvalidArgument, "tenant_id is required")
	}
	if req.EventType == "" {
		return nil, status.Error(codes.InvalidArgument, "event_type is required")
	}

	pref := &db.NotificationPreference{
		TenantID:  req.TenantId,
		Channel:   channelToString(req.Channel),
		EventType: req.EventType,
		Enabled:   req.Enabled,
		Config:    req.Config,
	}

	if err := s.store.UpsertPreference(ctx, pref); err != nil {
		s.logger.Error("UpsertPreference store error", "error", err)
		return nil, status.Errorf(codes.Internal, "upsert preference: %v", err)
	}

	s.logger.Info("notification preference updated",
		"tenant_id", req.TenantId, "channel", pref.Channel, "event_type", req.EventType)

	return &pb.UpdateNotificationPreferencesResponse{
		Preference: preferenceToProto(pref),
	}, nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

func channelToString(c pb.NotificationChannel) string {
	switch c {
	case pb.NotificationChannel_NOTIFICATION_CHANNEL_EMAIL:
		return "email"
	case pb.NotificationChannel_NOTIFICATION_CHANNEL_WEBHOOK:
		return "webhook"
	case pb.NotificationChannel_NOTIFICATION_CHANNEL_SLACK:
		return "slack"
	default:
		return "webhook"
	}
}

func channelFromString(s string) pb.NotificationChannel {
	switch s {
	case "email":
		return pb.NotificationChannel_NOTIFICATION_CHANNEL_EMAIL
	case "slack":
		return pb.NotificationChannel_NOTIFICATION_CHANNEL_SLACK
	default:
		return pb.NotificationChannel_NOTIFICATION_CHANNEL_WEBHOOK
	}
}

func statusFromString(s string) pb.NotificationStatus {
	switch s {
	case "pending":
		return pb.NotificationStatus_NOTIFICATION_STATUS_PENDING
	case "delivered":
		return pb.NotificationStatus_NOTIFICATION_STATUS_DELIVERED
	case "failed":
		return pb.NotificationStatus_NOTIFICATION_STATUS_FAILED
	case "retrying":
		return pb.NotificationStatus_NOTIFICATION_STATUS_RETRYING
	default:
		return pb.NotificationStatus_NOTIFICATION_STATUS_UNSPECIFIED
	}
}

func notificationToProto(n *db.Notification) *pb.Notification {
	p := &pb.Notification{
		Id:        n.ID,
		TenantId:  n.TenantID,
		Type:      n.Type,
		Recipient: n.Recipient,
		Channel:   channelFromString(n.Channel),
		Payload:   n.Payload,
		Status:    statusFromString(n.Status),
		Attempts:  int32(n.Attempts),
		CreatedAt: timestamppb.New(n.CreatedAt),
	}
	if n.LastAttemptAt != nil {
		p.LastAttemptAt = timestamppb.New(*n.LastAttemptAt)
	}
	return p
}

func preferenceToProto(p *db.NotificationPreference) *pb.NotificationPreference {
	return &pb.NotificationPreference{
		TenantId:  p.TenantID,
		Channel:   channelFromString(p.Channel),
		EventType: p.EventType,
		Enabled:   p.Enabled,
		Config:    p.Config,
	}
}
