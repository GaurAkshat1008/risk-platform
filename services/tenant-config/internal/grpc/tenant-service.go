package grpc

import (
    "context"
    "log/slog"

    pb "tenant-config/api/gen/tenant"
    "tenant-config/internal/cache"
    "tenant-config/internal/db"
    "tenant-config/internal/kafka"

    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"
    "google.golang.org/protobuf/types/known/timestamppb"
)

type TenantConfigService struct {
    pb.UnimplementedTenantConfigServiceServer
    store     *db.TenantStore
    cache     *cache.TenantCache
    publisher *kafka.TenantEventPublisher
    logger    *slog.Logger
}

func NewTenantConfigService(
    store *db.TenantStore,
    tenantCache *cache.TenantCache,
    publisher *kafka.TenantEventPublisher,
    logger *slog.Logger,
) *TenantConfigService {
    return &TenantConfigService{
        store:     store,
        cache:     tenantCache,
        publisher: publisher,
        logger:    logger,
    }
}

func (s *TenantConfigService) CreateTenant(ctx context.Context, req *pb.CreateTenantRequest) (*pb.CreateTenantResponse, error) {
    if req.Name == "" {
        return nil, status.Error(codes.InvalidArgument, "name is required")
    }

    var flags []db.FeatureFlag
    var meta map[string]string
    ruleSetID, workflowTemplateID := "", ""
    if req.Config != nil {
        ruleSetID = req.Config.RuleSetId
        workflowTemplateID = req.Config.WorkflowTemplateId
        meta = req.Config.Metadata
        for _, f := range req.Config.FeatureFlags {
            flags = append(flags, db.FeatureFlag{Key: f.Key, Enabled: f.Enabled, RolloutPercentage: f.RolloutPercentage})
        }
    }

    t, cfg, err := s.store.CreateTenant(ctx, req.Name, ruleSetID, workflowTemplateID, flags, meta)
    if err != nil {
        s.logger.Error("CreateTenant db error", "error", err)
        return nil, status.Errorf(codes.Internal, "create tenant: %v", err)
    }

    dbFlags, _ := s.store.GetFeatureFlags(ctx, t.ID)

    if err := s.cache.Set(ctx, toRecord(t, cfg, dbFlags)); err != nil {
        s.logger.Warn("cache set failed", "tenant_id", t.ID, "error", err)
    }
    if err := s.publisher.PublishTenantCreated(ctx, t.ID, t.Name); err != nil {
        s.logger.Warn("publish tenant.created failed", "tenant_id", t.ID, "error", err)
    }

    return &pb.CreateTenantResponse{Tenant: toProtoTenant(t, cfg, dbFlags)}, nil
}

func (s *TenantConfigService) GetTenantConfig(ctx context.Context, req *pb.GetTenantConfigRequest) (*pb.GetTenantConfigResponse, error) {
    if req.TenantId == "" {
        return nil, status.Error(codes.InvalidArgument, "tenant_id is required")
    }

    if rec, err := s.cache.Get(ctx, req.TenantId); err == nil && rec != nil {
        return &pb.GetTenantConfigResponse{Tenant: recordToProto(rec)}, nil
    }

    t, cfg, flags, err := s.store.GetTenant(ctx, req.TenantId)
    if err != nil {
        s.logger.Error("GetTenantConfig db error", "tenant_id", req.TenantId, "error", err)
        return nil, status.Errorf(codes.Internal, "get tenant: %v", err)
    }

    if err := s.cache.Set(ctx, toRecord(t, cfg, flags)); err != nil {
        s.logger.Warn("cache set failed", "tenant_id", req.TenantId, "error", err)
    }
    return &pb.GetTenantConfigResponse{Tenant: toProtoTenant(t, cfg, flags)}, nil
}

func (s *TenantConfigService) UpdateTenantRuleConfig(ctx context.Context, req *pb.UpdateTenantRuleConfigRequest) (*pb.UpdateTenantRuleConfigResponse, error) {
    if req.TenantId == "" || req.RuleSetId == "" {
        return nil, status.Error(codes.InvalidArgument, "tenant_id and rule_set_id are required")
    }

    t, cfg, flags, err := s.store.UpdateRuleConfig(ctx, req.TenantId, req.RuleSetId)
    if err != nil {
        s.logger.Error("UpdateTenantRuleConfig db error", "tenant_id", req.TenantId, "error", err)
        return nil, status.Errorf(codes.Internal, "update rule config: %v", err)
    }

    if err := s.cache.Set(ctx, toRecord(t, cfg, flags)); err != nil {
        s.logger.Warn("cache update failed", "tenant_id", req.TenantId, "error", err)
    }
    if err := s.publisher.PublishConfigUpdated(ctx, req.TenantId, "rule_config", cfg.Version); err != nil {
        s.logger.Warn("publish config.updated failed", "tenant_id", req.TenantId, "error", err)
    }
    return &pb.UpdateTenantRuleConfigResponse{Tenant: toProtoTenant(t, cfg, flags)}, nil
}

func (s *TenantConfigService) UpdateTenantWorkflowConfig(ctx context.Context, req *pb.UpdateTenantWorkflowConfigRequest) (*pb.UpdateTenantWorkflowConfigResponse, error) {
    if req.TenantId == "" || req.WorkflowTemplateId == "" {
        return nil, status.Error(codes.InvalidArgument, "tenant_id and workflow_template_id are required")
    }

    t, cfg, flags, err := s.store.UpdateWorkflowConfig(ctx, req.TenantId, req.WorkflowTemplateId)
    if err != nil {
        s.logger.Error("UpdateTenantWorkflowConfig db error", "tenant_id", req.TenantId, "error", err)
        return nil, status.Errorf(codes.Internal, "update workflow config: %v", err)
    }

    if err := s.cache.Set(ctx, toRecord(t, cfg, flags)); err != nil {
        s.logger.Warn("cache update failed", "tenant_id", req.TenantId, "error", err)
    }
    if err := s.publisher.PublishConfigUpdated(ctx, req.TenantId, "workflow_config", cfg.Version); err != nil {
        s.logger.Warn("publish config.updated failed", "tenant_id", req.TenantId, "error", err)
    }
    return &pb.UpdateTenantWorkflowConfigResponse{Tenant: toProtoTenant(t, cfg, flags)}, nil
}

func (s *TenantConfigService) GetFeatureFlags(ctx context.Context, req *pb.GetFeatureFlagsRequest) (*pb.GetFeatureFlagsResponse, error) {
    if req.TenantId == "" {
        return nil, status.Error(codes.InvalidArgument, "tenant_id is required")
    }

    if rec, err := s.cache.Get(ctx, req.TenantId); err == nil && rec != nil {
        var pbFlags []*pb.FeatureFlag
        for _, f := range rec.FeatureFlags {
            pbFlags = append(pbFlags, &pb.FeatureFlag{Key: f.Key, Enabled: f.Enabled, RolloutPercentage: f.RolloutPercentage})
        }
        return &pb.GetFeatureFlagsResponse{FeatureFlags: pbFlags}, nil
    }

    flags, err := s.store.GetFeatureFlags(ctx, req.TenantId)
    if err != nil {
        s.logger.Error("GetFeatureFlags db error", "tenant_id", req.TenantId, "error", err)
        return nil, status.Errorf(codes.Internal, "get feature flags: %v", err)
    }

    var pbFlags []*pb.FeatureFlag
    for _, f := range flags {
        pbFlags = append(pbFlags, &pb.FeatureFlag{Key: f.Key, Enabled: f.Enabled, RolloutPercentage: f.RolloutPercentage})
    }
    return &pb.GetFeatureFlagsResponse{FeatureFlags: pbFlags}, nil
}

// ---- conversion helpers ----

func toProtoTenant(t *db.Tenant, cfg *db.TenantConfig, flags []db.FeatureFlag) *pb.Tenant {
    var pbFlags []*pb.FeatureFlag
    for _, f := range flags {
        pbFlags = append(pbFlags, &pb.FeatureFlag{Key: f.Key, Enabled: f.Enabled, RolloutPercentage: f.RolloutPercentage})
    }
    return &pb.Tenant{
        Id:        t.ID,
        Name:      t.Name,
        Status:    toProtoStatus(t.Status),
        CreatedAt: timestamppb.New(t.CreatedAt),
        Config: &pb.TenantConfig{
            RuleSetId:          cfg.RuleSetId,
            WorkflowTemplateId: cfg.WorkflowTemplateID,
            FeatureFlags:       pbFlags,
            Metadata:           cfg.Metadata,
            Version:            cfg.Version,
        },
    }
}

func toProtoStatus(s db.TenantStatus) pb.TenantStatus {
    switch s {
    case db.TenantStatusActive:
        return pb.TenantStatus_TENANT_STATUS_ACTIVE
    case db.TenantStatusSuspended:
        return pb.TenantStatus_TENANT_STATUS_INACTIVE
    case db.TenantStatusOnboarding:
        return pb.TenantStatus_TENANT_STATUS_ONBOARDING
    default:
        return pb.TenantStatus_TENANT_STATUS_UNSPECIFIED
    }
}

func toRecord(t *db.Tenant, cfg *db.TenantConfig, flags []db.FeatureFlag) cache.TenantRecord {
    var fr []cache.FlagRecord
    for _, f := range flags {
        fr = append(fr, cache.FlagRecord{Key: f.Key, Enabled: f.Enabled, RolloutPercentage: f.RolloutPercentage})
    }
    return cache.TenantRecord{
        TenantID:           t.ID,
        Name:               t.Name,
        Status:             string(t.Status),
        CreatedAt:          t.CreatedAt,
        RuleSetID:          cfg.RuleSetId,
        WorkflowTemplateID: cfg.WorkflowTemplateID,
        Metadata:           cfg.Metadata,
        Version:            cfg.Version,
        FeatureFlags:       fr,
    }
}

func recordToProto(rec *cache.TenantRecord) *pb.Tenant {
    var pbFlags []*pb.FeatureFlag
    for _, f := range rec.FeatureFlags {
        pbFlags = append(pbFlags, &pb.FeatureFlag{Key: f.Key, Enabled: f.Enabled, RolloutPercentage: f.RolloutPercentage})
    }
    return &pb.Tenant{
        Id:        rec.TenantID,
        Name:      rec.Name,
        Status:    toProtoStatus(db.TenantStatus(rec.Status)),
        CreatedAt: timestamppb.New(rec.CreatedAt),
        Config: &pb.TenantConfig{
            RuleSetId:          rec.RuleSetID,
            WorkflowTemplateId: rec.WorkflowTemplateID,
            FeatureFlags:       pbFlags,
            Metadata:           rec.Metadata,
            Version:            rec.Version,
        },
    }
}