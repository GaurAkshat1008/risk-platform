package client

import (
	"context"
	"fmt"

	identity "graphql-bff/api/gen/identity"

	"google.golang.org/grpc"
)

type IdentityClient struct {
	stub identity.IdentityAccessServiceClient
}

func NewIdentityClient(conn *grpc.ClientConn) *IdentityClient {
	return &IdentityClient{stub: identity.NewIdentityAccessServiceClient(conn)}
}

func (c *IdentityClient) ValidateToken(ctx context.Context, accessToken string) (bool, string, *identity.Principal, error) {
	resp, err := c.stub.ValidateToken(ctx, &identity.ValidateTokenRequest{AccessToken: accessToken})
	if err != nil {
		return false, "", nil, fmt.Errorf("identity.ValidateToken: %w", err)
	}
	return resp.Valid, resp.Reason, resp.Principal, nil
}

func (c *IdentityClient) GetPermissions(ctx context.Context, accessToken, tenantID string) ([]string, error) {
	resp, err := c.stub.GetPermissions(ctx, &identity.GetPermissionsRequest{
		AccessToken: accessToken,
		TenantId:    tenantID,
	})
	if err != nil {
		return nil, fmt.Errorf("identity.GetPermissions: %w", err)
	}
	return resp.Permissions, nil
}
