package client

import (
	"context"
	"fmt"

	workflowpb "graphql-bff/api/gen/workflow"

	"google.golang.org/grpc"
)

type WorkflowClient struct {
	stub workflowpb.WorkflowServiceClient
}

func NewWorkflowClient(conn *grpc.ClientConn) *WorkflowClient {
	return &WorkflowClient{stub: workflowpb.NewWorkflowServiceClient(conn)}
}

func (c *WorkflowClient) GetTemplate(ctx context.Context, templateID, tenantID string) (*workflowpb.WorkflowTemplate, error) {
	ctx = attachToken(ctx)
	resp, err := c.stub.GetWorkflowTemplate(ctx, &workflowpb.GetWorkflowTemplateRequest{
		TemplateId: templateID,
		TenantId:   tenantID,
	})
	if err != nil {
		return nil, fmt.Errorf("workflow.GetWorkflowTemplate: %w", err)
	}
	return resp.Template, nil
}

func (c *WorkflowClient) ListTransitions(ctx context.Context, templateID, tenantID string) ([]*workflowpb.Transition, error) {
	ctx = attachToken(ctx)
	resp, err := c.stub.ListTransitions(ctx, &workflowpb.ListTransitionsRequest{
		TemplateId: templateID,
		TenantId:   tenantID,
	})
	if err != nil {
		return nil, fmt.Errorf("workflow.ListTransitions: %w", err)
	}
	return resp.Transitions, nil
}

func (c *WorkflowClient) CreateTemplate(ctx context.Context, req *workflowpb.CreateWorkflowTemplateRequest) (*workflowpb.WorkflowTemplate, error) {
	ctx = attachToken(ctx)
	resp, err := c.stub.CreateWorkflowTemplate(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("workflow.CreateWorkflowTemplate: %w", err)
	}
	return resp.Template, nil
}

func (c *WorkflowClient) UpdateTemplate(ctx context.Context, req *workflowpb.UpdateWorkflowTemplateRequest) (*workflowpb.WorkflowTemplate, error) {
	ctx = attachToken(ctx)
	resp, err := c.stub.UpdateWorkflowTemplate(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("workflow.UpdateWorkflowTemplate: %w", err)
	}
	return resp.Template, nil
}

func (c *WorkflowClient) EvaluateTransition(ctx context.Context, req *workflowpb.EvaluateTransitionRequest) (bool, string, error) {
	ctx = attachToken(ctx)
	resp, err := c.stub.EvaluateTransition(ctx, req)
	if err != nil {
		return false, "", fmt.Errorf("workflow.EvaluateTransition: %w", err)
	}
	return resp.Allowed, resp.Reason, nil
}
