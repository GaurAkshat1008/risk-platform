import { lazy, Suspense } from "react";
import { Navigate, Route, Routes } from "react-router-dom";
import { useAuth } from "@/hooks/useAuth";
import { defaultPortal } from "@/utils/roleHelpers";
import { RoleGate } from "@/components/layout/RoleGate";
import { PageSkeleton } from "@/components/common/PageSkeleton";
import { ROLES } from "@/utils/roleHelpers";

/* ---------- Merchant ---------- */
const PaymentSubmit = lazy(() => import("@/pages/merchant/PaymentSubmit"));
const PaymentLookup = lazy(() => import("@/pages/merchant/PaymentLookup"));
const DecisionList = lazy(() => import("@/pages/merchant/DecisionList"));
const DecisionDetail = lazy(() => import("@/pages/merchant/DecisionDetail"));

/* ---------- Analyst ---------- */
const CaseQueue = lazy(() => import("@/pages/analyst/CaseQueue"));
const CaseDetail = lazy(() => import("@/pages/analyst/CaseDetail"));

/* ---------- Ops ---------- */
const LogExplorer = lazy(() => import("@/pages/ops/LogExplorer"));
const TraceViewer = lazy(() => import("@/pages/ops/TraceViewer"));
const SLODashboard = lazy(() => import("@/pages/ops/SLODashboard"));
const AlertCenter = lazy(() => import("@/pages/ops/AlertCenter"));

/* ---------- Admin ---------- */
const TenantOverview = lazy(() => import("@/pages/admin/TenantOverview"));
const RulesManager = lazy(() => import("@/pages/admin/RulesManager"));
const WorkflowBuilder = lazy(() => import("@/pages/admin/WorkflowBuilder"));
const AuditViewer = lazy(() => import("@/pages/admin/AuditViewer"));
const NotificationsAdmin = lazy(
  () => import("@/pages/admin/NotificationsAdmin"),
);

function Fallback() {
  return <PageSkeleton />;
}

export function AppRoutes() {
  const { user } = useAuth();

  return (
    <Suspense fallback={<Fallback />}>
      <Routes>
        {/* Root redirect based on role */}
        <Route
          path="/"
          element={<Navigate to={defaultPortal(user.roles)} replace />}
        />

        {/* ---- Merchant Portal ---- */}
        <Route
          path="/merchant/submit"
          element={
            <RoleGate requires={[ROLES.MERCHANT, ROLES.PLATFORM_ADMIN]}>
              <PaymentSubmit />
            </RoleGate>
          }
        />
        <Route
          path="/merchant/lookup"
          element={
            <RoleGate requires={[ROLES.MERCHANT, ROLES.PLATFORM_ADMIN]}>
              <PaymentLookup />
            </RoleGate>
          }
        />
        <Route
          path="/merchant/decisions"
          element={
            <RoleGate requires={[ROLES.MERCHANT, ROLES.PLATFORM_ADMIN]}>
              <DecisionList />
            </RoleGate>
          }
        />
        <Route
          path="/merchant/decisions/:id"
          element={
            <RoleGate requires={[ROLES.MERCHANT, ROLES.PLATFORM_ADMIN]}>
              <DecisionDetail />
            </RoleGate>
          }
        />
        <Route
          path="/merchant"
          element={<Navigate to="/merchant/decisions" replace />}
        />

        {/* ---- Analyst Workspace ---- */}
        <Route
          path="/analyst/cases"
          element={
            <RoleGate requires={[ROLES.ANALYST, ROLES.PLATFORM_ADMIN]}>
              <CaseQueue />
            </RoleGate>
          }
        />
        <Route
          path="/analyst/cases/:id"
          element={
            <RoleGate requires={[ROLES.ANALYST, ROLES.PLATFORM_ADMIN]}>
              <CaseDetail />
            </RoleGate>
          }
        />
        <Route
          path="/analyst"
          element={<Navigate to="/analyst/cases" replace />}
        />

        {/* ---- Ops Dashboard ---- */}
        <Route
          path="/ops/logs"
          element={
            <RoleGate requires={[ROLES.OPS_ADMIN, ROLES.PLATFORM_ADMIN]}>
              <LogExplorer />
            </RoleGate>
          }
        />
        <Route
          path="/ops/traces"
          element={
            <RoleGate requires={[ROLES.OPS_ADMIN, ROLES.PLATFORM_ADMIN]}>
              <TraceViewer />
            </RoleGate>
          }
        />
        <Route
          path="/ops/slo"
          element={
            <RoleGate requires={[ROLES.OPS_ADMIN, ROLES.PLATFORM_ADMIN]}>
              <SLODashboard />
            </RoleGate>
          }
        />
        <Route
          path="/ops/alerts"
          element={
            <RoleGate requires={[ROLES.OPS_ADMIN, ROLES.PLATFORM_ADMIN]}>
              <AlertCenter />
            </RoleGate>
          }
        />
        <Route path="/ops" element={<Navigate to="/ops/slo" replace />} />

        {/* ---- Admin Console ---- */}
        <Route
          path="/admin/tenant"
          element={
            <RoleGate requires={[ROLES.TENANT_ADMIN, ROLES.PLATFORM_ADMIN]}>
              <TenantOverview />
            </RoleGate>
          }
        />
        <Route
          path="/admin/rules"
          element={
            <RoleGate requires={[ROLES.TENANT_ADMIN, ROLES.PLATFORM_ADMIN]}>
              <RulesManager />
            </RoleGate>
          }
        />
        <Route
          path="/admin/workflows"
          element={
            <RoleGate requires={[ROLES.TENANT_ADMIN, ROLES.PLATFORM_ADMIN]}>
              <WorkflowBuilder />
            </RoleGate>
          }
        />
        <Route
          path="/admin/audit"
          element={
            <RoleGate requires={[ROLES.TENANT_ADMIN, ROLES.PLATFORM_ADMIN]}>
              <AuditViewer />
            </RoleGate>
          }
        />
        <Route
          path="/admin/notifications"
          element={
            <RoleGate requires={[ROLES.TENANT_ADMIN, ROLES.PLATFORM_ADMIN]}>
              <NotificationsAdmin />
            </RoleGate>
          }
        />
        <Route
          path="/admin"
          element={<Navigate to="/admin/tenant" replace />}
        />

        {/* Catch-all */}
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </Suspense>
  );
}
