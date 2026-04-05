export const ROLES = {
  PLATFORM_ADMIN: 'platform_admin',
  TENANT_ADMIN: 'tenant_admin',
  ANALYST: 'analyst',
  MERCHANT: 'merchant_user',
  OPS_ADMIN: 'ops_admin',
} as const;

export function hasRole(roles: string[], role: string): boolean {
  return roles.includes(role);
}

export function hasAnyRole(roles: string[], ...required: string[]): boolean {
  return required.some((r) => roles.includes(r));
}

export function isAdmin(roles: string[]): boolean {
  return roles.includes(ROLES.PLATFORM_ADMIN);
}

/** Return the primary portal path for a given role set (first match wins). */
export function defaultPortal(roles: string[]): string {
  if (hasAnyRole(roles, ROLES.PLATFORM_ADMIN, ROLES.TENANT_ADMIN)) return '/admin';
  if (hasRole(roles, ROLES.ANALYST)) return '/analyst';
  if (hasRole(roles, ROLES.OPS_ADMIN)) return '/ops';
  if (hasRole(roles, ROLES.MERCHANT)) return '/merchant';
  return '/merchant';
}
