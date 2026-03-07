package grpc

import (
	"context"
	"log/slog"

	identitypb "identity-access/api/gen/identity"
	"identity-access/internal/auth"
	"identity-access/internal/kafka"
	"identity-access/internal/keycloak"
	"identity-access/internal/rbac"
	"identity-access/internal/telemetry"

	"go.opentelemetry.io/otel/attribute"
	otelcodes "go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	grpccodes "google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var highStakesActions = map[string]bool{
	"decision:override": true,
	"case:escalate":     true,
	"workflow:approve":  true,
}

type IdentityServiceServer struct {
	identitypb.UnimplementedIdentityAccessServiceServer
	validator      *auth.Validator
	rbac           *rbac.Evaluator
	keycloakClient *keycloak.Client
	publisher      *kafka.AuthEventPublisher
	metrics        *telemetry.Metrics
}

func NewIdentityServiceServer(
	validator *auth.Validator,
	rbacEval *rbac.Evaluator,
	keycloakClient *keycloak.Client,
	publisher *kafka.AuthEventPublisher,
	metrics *telemetry.Metrics,
) *IdentityServiceServer {
	return &IdentityServiceServer{
		validator:      validator,
		rbac:           rbacEval,
		keycloakClient: keycloakClient,
		publisher:      publisher,
		metrics:        metrics,
	}
}

func (s *IdentityServiceServer) ValidateToken(ctx context.Context, req *identitypb.ValidateTokenRequest) (*identitypb.ValidateTokenResponse, error) {
	tracer := telemetry.Tracer("identity.ValidateToken")
	ctx, span := tracer.Start(ctx, "ValidateToken")
	defer span.End()

	traceID := span.SpanContext().TraceID().String()

	p, err := s.validator.ValidateAccessToken(ctx, req.GetAccessToken())
	if err != nil {
		span.SetStatus(otelcodes.Error, err.Error())
		span.SetAttributes(attribute.String("auth.result", "rejected"))
		_ = s.publisher.PublishTokenRejected(ctx, err.Error(), traceID)
		return &identitypb.ValidateTokenResponse{
			Valid:  false,
			Reason: err.Error(),
		}, nil
	}

	span.SetStatus(otelcodes.Ok, "")
	span.SetAttributes(
		attribute.String("auth.user_id", p.UserID),
		attribute.String("auth.tenant_id", p.TenantID),
		attribute.String("auth.result", "valid"),
	)
	_ = s.publisher.PublishTokenValidated(ctx, p.UserID, p.TenantID, traceID)

	slog.InfoContext(ctx, "token validated",
		"user_id", p.UserID,
		"tenant_id", p.TenantID,
		"trace_id", traceID,
	)

	return &identitypb.ValidateTokenResponse{
		Valid:  true,
		Reason: "ok",
		Principal: &identitypb.Principal{
			UserId:   p.UserID,
			TenantId: p.TenantID,
			Roles:    p.Roles,
			ExpUnix:  p.ExpiresAt.Unix(),
			Issuer:   p.Issuer,
			Audience: p.Audience,
		},
	}, nil
}

func (s *IdentityServiceServer) Authorize(ctx context.Context, req *identitypb.AuthorizeRequest) (*identitypb.AuthorizeResponse, error) {
	tracer := telemetry.Tracer("identity.Authorize")
	ctx, span := tracer.Start(ctx, "Authorize",
		trace.WithAttributes(attribute.String("auth.action", req.GetAction())),
	)
	defer span.End()

	traceID := span.SpanContext().TraceID().String()

	// step 1: validate JWT
	p, err := s.validator.ValidateAccessToken(ctx, req.GetAccessToken())
	if err != nil {
		span.SetStatus(otelcodes.Error, err.Error())
		_ = s.publisher.PublishTokenRejected(ctx, err.Error(), traceID)
		return nil, status.Error(grpccodes.Unauthenticated, "invalid access token: "+err.Error())
	}

	span.SetAttributes(
		attribute.String("auth.user_id", p.UserID),
		attribute.String("auth.tenant_id", p.TenantID),
	)

	// step 2: live Keycloak checks for high-stakes actions only
	if highStakesActions[req.GetAction()] {
		_, kcSpan := tracer.Start(ctx, "Authorize.KeycloakChecks")

		enabled, err := s.keycloakClient.IsUserEnabled(ctx, p.UserID)
		if err != nil {
			kcSpan.SetStatus(otelcodes.Error, err.Error())
			kcSpan.End()
			return nil, status.Error(grpccodes.Internal, "failed to check user status")
		}
		if !enabled {
			kcSpan.SetAttributes(attribute.Bool("auth.account_enabled", false))
			kcSpan.End()
			_ = s.publisher.PublishAccountDisabled(ctx, p.UserID, p.TenantID, traceID)
			return &identitypb.AuthorizeResponse{
				Allowed: false,
				Reason:  "user account is disabled",
			}, nil
		}

		liveTenantID, err := s.keycloakClient.GetUserTenantID(ctx, p.UserID)
		if err != nil {
			kcSpan.SetStatus(otelcodes.Error, err.Error())
			kcSpan.End()
			return nil, status.Error(grpccodes.Internal, "tenant verification failed")
		}
		if liveTenantID != p.TenantID {
			kcSpan.SetAttributes(attribute.Bool("auth.tenant_match", false))
			kcSpan.End()
			return &identitypb.AuthorizeResponse{
				Allowed: false,
				Reason:  "tenant mismatch - token may be stale",
			}, nil
		}

		kcSpan.SetAttributes(attribute.Bool("auth.account_enabled", true))
		kcSpan.End()
	}

	// step 3: RBAC decision
	allowed, reason, effectivePerms := s.rbac.Can(p, req.GetAction(), req.GetResourceTenantId())

	span.SetAttributes(
		attribute.Bool("auth.allowed", allowed),
		attribute.String("auth.reason", reason),
	)
	if allowed {
		span.SetStatus(otelcodes.Ok, "")
	} else {
		span.SetStatus(otelcodes.Error, reason)
	}

	_ = s.publisher.PublishAuthzDecision(ctx, p.UserID, p.TenantID, req.GetAction(), allowed, reason, traceID)

	slog.InfoContext(ctx, "authorize decision",
		"user_id", p.UserID,
		"tenant_id", p.TenantID,
		"action", req.GetAction(),
		"allowed", allowed,
		"reason", reason,
		"trace_id", traceID,
	)

	return &identitypb.AuthorizeResponse{
		Allowed:              allowed,
		Reason:               reason,
		EffectivePermissions: effectivePerms,
	}, nil
}

func (s *IdentityServiceServer) GetPermissions(ctx context.Context, req *identitypb.GetPermissionsRequest) (*identitypb.GetPermissionsResponse, error) {
	tracer := telemetry.Tracer("identity.GetPermissions")
	ctx, span := tracer.Start(ctx, "GetPermissions")
	defer span.End()

	p, err := s.validator.ValidateAccessToken(ctx, req.GetAccessToken())
	if err != nil {
		span.SetStatus(otelcodes.Error, err.Error())
		return nil, status.Error(grpccodes.Unauthenticated, "invalid access token: "+err.Error())
	}

	if req.GetTenantId() != "" && req.GetTenantId() != p.TenantID {
		span.SetStatus(otelcodes.Error, "tenant_mismatch")
		return nil, status.Error(grpccodes.PermissionDenied, "tenant_mismatch")
	}

	perms := s.rbac.EffectivePermissions(p)

	span.SetStatus(otelcodes.Ok, "")
	span.SetAttributes(
		attribute.String("auth.user_id", p.UserID),
		attribute.String("auth.tenant_id", p.TenantID),
		attribute.Int("auth.permissions_count", len(perms)),
	)

	return &identitypb.GetPermissionsResponse{
		Permissions: perms,
	}, nil
}
