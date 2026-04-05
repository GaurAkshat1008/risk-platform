import { useMemo } from "react";
import { useNavigate, useLocation } from "react-router-dom";
import { Layout, Menu, Typography } from "antd";
import type { ItemType } from "antd/es/menu/interface";
import {
  SendOutlined,
  SearchOutlined,
  CheckCircleOutlined,
  FileSearchOutlined,
  CodeOutlined,
  DashboardOutlined,
  AlertOutlined,
  TeamOutlined,
  ToolOutlined,
  BranchesOutlined,
  AuditOutlined,
  BellOutlined,
  ApartmentOutlined,
} from "@ant-design/icons";
import { useAuth } from "@/hooks/useAuth";
import { ROLES } from "@/utils/roleHelpers";

const { Sider } = Layout;

interface Props {
  collapsed: boolean;
  onCollapse: (collapsed: boolean) => void;
}

export function Sidebar({ collapsed, onCollapse }: Props) {
  const { hasAnyRole } = useAuth();
  const navigate = useNavigate();
  const location = useLocation();

  const items = useMemo<ItemType[]>(() => {
    const result: ItemType[] = [];

    if (hasAnyRole(ROLES.MERCHANT, ROLES.PLATFORM_ADMIN)) {
      result.push({
        key: "merchant",
        label: "Merchant",
        type: "group",
        children: [
          {
            key: "/merchant/submit",
            icon: <SendOutlined />,
            label: "Submit Payment",
          },
          {
            key: "/merchant/lookup",
            icon: <SearchOutlined />,
            label: "Lookup",
          },
          {
            key: "/merchant/decisions",
            icon: <CheckCircleOutlined />,
            label: "Decisions",
          },
        ],
      });
    }

    if (hasAnyRole(ROLES.ANALYST, ROLES.PLATFORM_ADMIN)) {
      result.push({
        key: "analyst",
        label: "Analyst",
        type: "group",
        children: [
          {
            key: "/analyst/cases",
            icon: <FileSearchOutlined />,
            label: "Case Queue",
          },
        ],
      });
    }

    if (hasAnyRole(ROLES.OPS_ADMIN, ROLES.PLATFORM_ADMIN)) {
      result.push({
        key: "ops",
        label: "Ops",
        type: "group",
        children: [
          { key: "/ops/logs", icon: <CodeOutlined />, label: "Log Explorer" },
          { key: "/ops/traces", icon: <ApartmentOutlined />, label: "Traces" },
          {
            key: "/ops/slo",
            icon: <DashboardOutlined />,
            label: "SLO Dashboard",
          },
          { key: "/ops/alerts", icon: <AlertOutlined />, label: "Alerts" },
        ],
      });
    }

    if (hasAnyRole(ROLES.TENANT_ADMIN, ROLES.PLATFORM_ADMIN)) {
      result.push({
        key: "admin",
        label: "Admin",
        type: "group",
        children: [
          {
            key: "/admin/tenant",
            icon: <TeamOutlined />,
            label: "Tenant Config",
          },
          { key: "/admin/rules", icon: <ToolOutlined />, label: "Rules" },
          {
            key: "/admin/workflows",
            icon: <BranchesOutlined />,
            label: "Workflows",
          },
          {
            key: "/admin/audit",
            icon: <AuditOutlined />,
            label: "Audit Trail",
          },
          {
            key: "/admin/notifications",
            icon: <BellOutlined />,
            label: "Notifications",
          },
        ],
      });
    }

    return result;
  }, [hasAnyRole]);

  return (
    <Sider
      collapsible
      collapsed={collapsed}
      onCollapse={onCollapse}
      width={220}
      collapsedWidth={56}
      theme="dark"
      style={{
        overflow: "auto",
        height: "100vh",
        position: "fixed",
        left: 0,
        top: 0,
        bottom: 0,
        zIndex: 10,
      }}
    >
      <div
        style={{
          height: 48,
          display: "flex",
          alignItems: "center",
          justifyContent: collapsed ? "center" : "flex-start",
          padding: collapsed ? 0 : "0 20px",
          borderBottom: "1px solid rgba(255,255,255,0.06)",
        }}
      >
        <Typography.Text
          strong
          style={{ color: "#e2e8f0", fontSize: 15, whiteSpace: "nowrap" }}
        >
          {collapsed ? "RC" : "RiskCore"}
        </Typography.Text>
      </div>

      <Menu
        theme="dark"
        mode="inline"
        selectedKeys={[location.pathname]}
        items={items}
        onClick={({ key }) => navigate(key)}
        style={{ borderRight: 0, marginTop: 8 }}
      />
    </Sider>
  );
}
