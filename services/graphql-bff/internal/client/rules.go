package client

import (
	"context"
	"encoding/json"
	"fmt"

	rulesengine "graphql-bff/api/gen/rules-engine"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/structpb"
)

type RulesClient struct {
	stub rulesengine.RulesEngineServiceClient
}

func NewRulesClient(conn *grpc.ClientConn) *RulesClient {
	return &RulesClient{stub: rulesengine.NewRulesEngineServiceClient(conn)}
}

func (c *RulesClient) ListRules(ctx context.Context, tenantID string, includeDisabled bool) ([]*rulesengine.Rule, error) {
	ctx = attachToken(ctx)
	resp, err := c.stub.ListRules(ctx, &rulesengine.ListRulesRequest{
		TenantId:        tenantID,
		IncludeDisabled: includeDisabled,
	})
	if err != nil {
		return nil, fmt.Errorf("rules.ListRules: %w", err)
	}
	return resp.Rules, nil
}

func (c *RulesClient) CreateRule(ctx context.Context, tenantID, name, expression string, action rulesengine.RuleAction, priority int32) (*rulesengine.Rule, error) {
	ctx = attachToken(ctx)
	expr, err := parseExpression(expression)
	if err != nil {
		return nil, fmt.Errorf("parse expression: %w", err)
	}
	resp, err := c.stub.CreateRule(ctx, &rulesengine.CreateRuleRequest{
		TenantId:   tenantID,
		Name:       name,
		Expression: expr,
		Action:     action,
		Priority:   priority,
	})
	if err != nil {
		return nil, fmt.Errorf("rules.CreateRule: %w", err)
	}
	return resp.Rule, nil
}

func (c *RulesClient) UpdateRule(ctx context.Context, ruleID, tenantID, expression string, action rulesengine.RuleAction, priority int32, enabled bool) (*rulesengine.Rule, error) {
	ctx = attachToken(ctx)
	expr, err := parseExpression(expression)
	if err != nil {
		return nil, fmt.Errorf("parse expression: %w", err)
	}
	resp, err := c.stub.UpdateRule(ctx, &rulesengine.UpdateRuleRequest{
		RuleId:     ruleID,
		TenantId:   tenantID,
		Expression: expr,
		Action:     action,
		Priority:   priority,
		Enabled:    enabled,
	})
	if err != nil {
		return nil, fmt.Errorf("rules.UpdateRule: %w", err)
	}
	return resp.Rule, nil
}

func (c *RulesClient) DeleteRule(ctx context.Context, ruleID, tenantID string) (bool, error) {
	ctx = attachToken(ctx)
	resp, err := c.stub.DeleteRule(ctx, &rulesengine.DeleteRuleRequest{
		RuleId:   ruleID,
		TenantId: tenantID,
	})
	if err != nil {
		return false, fmt.Errorf("rules.DeleteRule: %w", err)
	}
	return resp.Success, nil
}

func (c *RulesClient) SimulateRule(ctx context.Context, req *rulesengine.SimulateRuleRequest) (*rulesengine.RuleResult, error) {
	ctx = attachToken(ctx)
	resp, err := c.stub.SimulateRule(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("rules.SimulateRule: %w", err)
	}
	return resp.Result, nil
}

// parseExpression converts a JSON string into a protobuf Struct.
func parseExpression(jsonStr string) (*structpb.Struct, error) {
	var m map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &m); err != nil {
		return nil, fmt.Errorf("invalid JSON expression: %w", err)
	}
	s, err := structpb.NewStruct(m)
	if err != nil {
		return nil, fmt.Errorf("convert to struct: %w", err)
	}
	return s, nil
}
