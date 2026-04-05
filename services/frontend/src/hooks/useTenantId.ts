import { useAuth } from './useAuth';

/**
 * Returns the current user's tenant ID from the Keycloak token.
 * All GraphQL queries inject this automatically.
 */
export function useTenantId(): string {
  const { user } = useAuth();
  return user.tenantId;
}
