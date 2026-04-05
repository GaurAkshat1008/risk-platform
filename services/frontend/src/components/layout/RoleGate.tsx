import type { ReactNode } from "react";
import { useAuth } from "@/hooks/useAuth";

interface Props {
  /** One or more roles that grant access. */
  requires: string | string[];
  children: ReactNode;
  /** Shown when the user lacks access (default: null). */
  fallback?: ReactNode;
}

/** Render children only if the current user has any of the required roles. */
export function RoleGate({ requires, children, fallback = null }: Props) {
  const { hasAnyRole, isAdmin } = useAuth();
  const roles = Array.isArray(requires) ? requires : [requires];
  if (isAdmin || hasAnyRole(...roles)) return <>{children}</>;
  return <>{fallback}</>;
}
