package keycloak

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// GetUserByID fetches a user profile from Keycloak by their user ID (the sub claim from JWT)
func (c *Client) GetUserByID(ctx context.Context, userID string) (*UserRepresentation, error) {
	path := fmt.Sprintf("/users/%s", userID)

	resp, err := c.doAdminRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("user not found: %s", userID)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get user by id unexpected status: %d", resp.StatusCode)
	}

	var user UserRepresentation
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("decode user response: %w", err)
	}
	return &user, nil
}

// GetUserRealmRoles fetches the live realm roles currently assigned to a user in Keycloak.
// This reflects the current state, not what was snapshotted in the JWT at login time.
// Use for high-stakes authorization checks (e.g. decision overrides, escalations).
func (c *Client) GetUserRealmRoles(ctx context.Context, userID string) ([]RoleRepresentation, error) {
	path := fmt.Sprintf("/users/%s/role-mappings/realm", userID)

	resp, err := c.doAdminRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("get user realm roles: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("user not found: %s", userID)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get user realm roles unexpected status: %d", resp.StatusCode)
	}

	var roles []RoleRepresentation
	if err := json.NewDecoder(resp.Body).Decode(&roles); err != nil {
		return nil, fmt.Errorf("decode roles response: %w", err)
	}
	return roles, nil
}

// IsUserEnabled checks whether the user account is currently active in Keycloak.
// A JWT can still be valid even if the account was disabled after login.
// Use before allowing sensitive actions to ensure account has not been revoked.
func (c *Client) IsUserEnabled(ctx context.Context, userID string) (bool, error) {
	user, err := c.GetUserByID(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("is user enabled: %w", err)
	}
	return user.Enabled, nil
}

// GetUserTenantID reads the tenant_id attribute from the user's Keycloak profile.
// Use as a second-factor check against the tenant_id claim in the JWT
// for high-stakes operations where stale token claims are a concern.
func (c *Client) GetUserTenantID(ctx context.Context, userID string) (string, error) {
	user, err := c.GetUserByID(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("get user tenant id: %w", err)
	}

	tenantIDs, ok := user.Attributes["tenant_id"]
	if !ok || len(tenantIDs) == 0 {
		return "", fmt.Errorf("tenant_id attribute missing for user: %s", userID)
	}

	return tenantIDs[0], nil
}