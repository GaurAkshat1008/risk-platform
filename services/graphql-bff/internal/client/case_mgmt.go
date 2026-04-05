package client

import (
	"context"
	"fmt"

	casemanagementpb "graphql-bff/api/gen/case-management"

	"google.golang.org/grpc"
)

type CaseManagementClient struct {
	stub casemanagementpb.CaseManagementServiceClient
}

func NewCaseManagementClient(conn *grpc.ClientConn) *CaseManagementClient {
	return &CaseManagementClient{stub: casemanagementpb.NewCaseManagementServiceClient(conn)}
}

func (c *CaseManagementClient) GetCase(ctx context.Context, caseID string) (*casemanagementpb.Case, error) {
	ctx = attachToken(ctx)
	resp, err := c.stub.GetCase(ctx, &casemanagementpb.GetCaseRequest{CaseId: caseID})
	if err != nil {
		return nil, fmt.Errorf("case.GetCase: %w", err)
	}
	return resp.Case, nil
}

func (c *CaseManagementClient) ListCases(ctx context.Context, req *casemanagementpb.ListCasesRequest) ([]*casemanagementpb.Case, string, error) {
	ctx = attachToken(ctx)
	resp, err := c.stub.ListCases(ctx, req)
	if err != nil {
		return nil, "", fmt.Errorf("case.ListCases: %w", err)
	}
	return resp.Cases, resp.NextPageToken, nil
}

func (c *CaseManagementClient) CreateCase(ctx context.Context, req *casemanagementpb.CreateCaseRequest) (*casemanagementpb.Case, error) {
	ctx = attachToken(ctx)
	resp, err := c.stub.CreateCase(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("case.CreateCase: %w", err)
	}
	return resp.Case, nil
}

func (c *CaseManagementClient) AssignCase(ctx context.Context, req *casemanagementpb.AssignCaseRequest) (*casemanagementpb.Case, error) {
	ctx = attachToken(ctx)
	resp, err := c.stub.AssignCase(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("case.AssignCase: %w", err)
	}
	return resp.Case, nil
}

func (c *CaseManagementClient) UpdateCaseStatus(ctx context.Context, req *casemanagementpb.UpdateCaseStatusRequest) (*casemanagementpb.Case, error) {
	ctx = attachToken(ctx)
	resp, err := c.stub.UpdateCaseStatus(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("case.UpdateCaseStatus: %w", err)
	}
	return resp.Case, nil
}

func (c *CaseManagementClient) EscalateCase(ctx context.Context, req *casemanagementpb.EscalateCaseRequest) (*casemanagementpb.Case, error) {
	ctx = attachToken(ctx)
	resp, err := c.stub.EscalateCase(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("case.EscalateCase: %w", err)
	}
	return resp.Case, nil
}
