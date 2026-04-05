import { useCallback, useMemo } from 'react';
import keycloak from '@/keycloak';

export interface AuthUser {
  userId: string;
  username: string;
  email: string;
  roles: string[];
  tenantId: string;
}

export function useAuth() {
  const tokenParsed = keycloak.tokenParsed as Record<string, unknown> | undefined;

  const user = useMemo<AuthUser>(() => {
    if (!tokenParsed) {
      return { userId: '', username: '', email: '', roles: [], tenantId: '' };
    }
    const realmRoles =
      (tokenParsed.realm_access as { roles?: string[] })?.roles ?? [];
    return {
      userId: (tokenParsed.sub as string) ?? '',
      username: (tokenParsed.preferred_username as string) ?? '',
      email: (tokenParsed.email as string) ?? '',
      roles: realmRoles,
      tenantId: (tokenParsed.tenant_id as string) ?? '',
    };
  }, [tokenParsed]);

  const hasRole = useCallback(
    (role: string) => user.roles.includes(role),
    [user.roles],
  );

  const hasAnyRole = useCallback(
    (...roles: string[]) => roles.some((r) => user.roles.includes(r)),
    [user.roles],
  );

  const isAdmin = useMemo(
    () => user.roles.includes('platform_admin'),
    [user.roles],
  );

  const logout = useCallback(() => {
    keycloak.logout();
  }, []);

  return { user, hasRole, hasAnyRole, isAdmin, logout, authenticated: keycloak.authenticated ?? false };
}
