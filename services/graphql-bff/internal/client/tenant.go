package client

import (
	"context"
	"fmt"

	tenant "graphql-bff/api/gen/tenant"

	"google.golang.org/grpc"
)

type TenantClient struct {
	stub tenant.TenantConfigServiceClient
}

func NewTenantClient(conn *grpc.ClientConn) *TenantClient {
	return &TenantClient{stub: tenant.NewTenantConfigServiceClient(conn)}
}

func (c *TenantClient) CreateTenant(ctx context.Context, name string, config *tenant.TenantConfig) (*tenant.Tenant, error) {
	ctx = attachToken(ctx)
	resp, err := c.stub.CreateTenant(ctx, &tenant.CreateTenantRequest{
		Name:   name,
		Config: config,
	})
	if err != nil {
		return nil, fmt.Errorf("tenant.CreateTenant: %w", err)
	}
	return resp.Tenant, nil
}

func (c *TenantClient) GetTenantConfig(ctx context.Context, tenantID string) (*tenant.Tenant, error) {
	ctx = attachToken(ctx)
	resp, err := c.stub.GetTenantConfig(ctx, &tenant.GetTenantConfigRequest{TenantId: tenantID})
	if err != nil {
		return nil, fmt.Errorf("tenant.GetTenantConfig: %w", err)
	}
	return resp.Tenant, nil
}

func (c *TenantClient) UpdateTenantRuleConfig(ctx context.Context, tenantID, ruleSetID string) (*tenant.Tenant, error) {
	ctx = attachToken(ctx)
	resp, err := c.stub.UpdateTenantRuleConfig(ctx, &tenant.UpdateTenantRuleConfigRequest{
		TenantId:  tenantID,
		RuleSetId: ruleSetID,
	})
	if err != nil {
		return nil, fmt.Errorf("tenant.UpdateTenantRuleConfig: %w", err)
	}
	return resp.Tenant, nil
}

func (c *TenantClient) UpdateTenantWorkflowConfig(ctx context.Context, tenantID, workflowTemplateID string) (*tenant.Tenant, error) {
	ctx = attachToken(ctx)
	resp, err := c.stub.UpdateTenantWorkflowConfig(ctx, &tenant.UpdateTenantWorkflowConfigRequest{
		TenantId:           tenantID,
		WorkflowTemplateId: workflowTemplateID,
	})
	if err != nil {
		return nil, fmt.Errorf("tenant.UpdateTenantWorkflowConfig: %w", err)
	}
	return resp.Tenant, nil
}

func (c *TenantClient) GetFeatureFlags(ctx context.Context, tenantID string) ([]*tenant.FeatureFlag, error) {
	ctx = attachToken(ctx)
	resp, err := c.stub.GetFeatureFlags(ctx, &tenant.GetFeatureFlagsRequest{TenantId: tenantID})
	if err != nil {
		return nil, fmt.Errorf("tenant.GetFeatureFlags: %w", err)
	}
	return resp.FeatureFlags, nil
}
