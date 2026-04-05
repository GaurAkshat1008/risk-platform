import { useCallback, useEffect, useMemo, useState } from "react";
import { Input, List, Modal, Tag, Typography } from "antd";
import { SearchOutlined } from "@ant-design/icons";
import { useNavigate } from "react-router-dom";
import { useAuth } from "@/hooks/useAuth";
import { ROLES } from "@/utils/roleHelpers";

interface PaletteEntry {
  label: string;
  path: string;
  section: string;
  roles: string[];
  keywords: string[];
}

const entries: PaletteEntry[] = [
  // Merchant
  {
    label: "Submit Payment",
    path: "/merchant/submit",
    section: "Merchant",
    roles: [ROLES.MERCHANT, ROLES.PLATFORM_ADMIN],
    keywords: ["ingest", "payment", "transaction"],
  },
  {
    label: "Lookup Payment",
    path: "/merchant/lookup",
    section: "Merchant",
    roles: [ROLES.MERCHANT, ROLES.PLATFORM_ADMIN],
    keywords: ["search", "find", "idempotency"],
  },
  {
    label: "Decisions",
    path: "/merchant/decisions",
    section: "Merchant",
    roles: [ROLES.MERCHANT, ROLES.PLATFORM_ADMIN],
    keywords: ["approve", "reject", "review", "block"],
  },
  // Analyst
  {
    label: "Case Queue",
    path: "/analyst/cases",
    section: "Analyst",
    roles: [ROLES.ANALYST, ROLES.PLATFORM_ADMIN],
    keywords: ["case", "queue", "review", "sla"],
  },
  // Ops
  {
    label: "Log Explorer",
    path: "/ops/logs",
    section: "Ops",
    roles: [ROLES.OPS_ADMIN, ROLES.PLATFORM_ADMIN],
    keywords: ["log", "search", "error", "debug"],
  },
  {
    label: "Trace Viewer",
    path: "/ops/traces",
    section: "Ops",
    roles: [ROLES.OPS_ADMIN, ROLES.PLATFORM_ADMIN],
    keywords: ["trace", "span", "jaeger", "latency"],
  },
  {
    label: "SLO Dashboard",
    path: "/ops/slo",
    section: "Ops",
    roles: [ROLES.OPS_ADMIN, ROLES.PLATFORM_ADMIN],
    keywords: ["slo", "availability", "p99", "error rate"],
  },
  {
    label: "Alert Center",
    path: "/ops/alerts",
    section: "Ops",
    roles: [ROLES.OPS_ADMIN, ROLES.PLATFORM_ADMIN],
    keywords: ["alert", "firing", "severity"],
  },
  // Admin
  {
    label: "Tenant Config",
    path: "/admin/tenant",
    section: "Admin",
    roles: [ROLES.TENANT_ADMIN, ROLES.PLATFORM_ADMIN],
    keywords: ["tenant", "config", "feature flag"],
  },
  {
    label: "Rules Manager",
    path: "/admin/rules",
    section: "Admin",
    roles: [ROLES.TENANT_ADMIN, ROLES.PLATFORM_ADMIN],
    keywords: ["rule", "create", "simulate"],
  },
  {
    label: "Workflow Builder",
    path: "/admin/workflows",
    section: "Admin",
    roles: [ROLES.TENANT_ADMIN, ROLES.PLATFORM_ADMIN],
    keywords: ["workflow", "template", "transition"],
  },
  {
    label: "Audit Viewer",
    path: "/admin/audit",
    section: "Admin",
    roles: [ROLES.TENANT_ADMIN, ROLES.PLATFORM_ADMIN],
    keywords: ["audit", "trail", "chain", "integrity"],
  },
  {
    label: "Notifications",
    path: "/admin/notifications",
    section: "Admin",
    roles: [ROLES.TENANT_ADMIN, ROLES.PLATFORM_ADMIN],
    keywords: ["notification", "email", "webhook"],
  },
];

const sectionColors: Record<string, string> = {
  Merchant: "blue",
  Analyst: "orange",
  Ops: "purple",
  Admin: "green",
};

export function CommandPalette() {
  const [open, setOpen] = useState(false);
  const [search, setSearch] = useState("");
  const navigate = useNavigate();
  const { user } = useAuth();

  // Cmd+K / Ctrl+K to toggle
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === "k") {
        e.preventDefault();
        setOpen((v) => !v);
        setSearch("");
      }
    };
    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
  }, []);

  const filtered = useMemo(() => {
    const accessible = entries.filter(
      (e) =>
        user.roles.includes(ROLES.PLATFORM_ADMIN) ||
        e.roles.some((r) => user.roles.includes(r)),
    );
    if (!search.trim()) return accessible;
    const q = search.toLowerCase();
    return accessible.filter(
      (e) =>
        e.label.toLowerCase().includes(q) ||
        e.section.toLowerCase().includes(q) ||
        e.keywords.some((k) => k.includes(q)),
    );
  }, [search, user.roles]);

  const go = useCallback(
    (path: string) => {
      navigate(path);
      setOpen(false);
      setSearch("");
    },
    [navigate],
  );

  return (
    <Modal
      open={open}
      onCancel={() => setOpen(false)}
      footer={null}
      closable={false}
      width={520}
      styles={{ body: { padding: 0 } }}
      style={{ top: "15vh" }}
    >
      <div
        style={{
          padding: "12px 16px",
          borderBottom: "1px solid var(--ant-color-border)",
        }}
      >
        <Input
          prefix={<SearchOutlined />}
          placeholder="Jump to page…"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          variant="borderless"
          size="large"
          autoFocus
          onKeyDown={(e) => {
            if (e.key === "Enter" && filtered.length > 0) {
              go(filtered[0].path);
            }
          }}
        />
      </div>
      <List
        size="small"
        dataSource={filtered}
        style={{ maxHeight: 360, overflow: "auto" }}
        renderItem={(item) => (
          <List.Item
            onClick={() => go(item.path)}
            style={{ cursor: "pointer", padding: "8px 20px" }}
          >
            <div
              style={{
                display: "flex",
                alignItems: "center",
                gap: 8,
                width: "100%",
              }}
            >
              <Tag
                color={sectionColors[item.section]}
                style={{ fontSize: 10, margin: 0 }}
              >
                {item.section}
              </Tag>
              <Typography.Text>{item.label}</Typography.Text>
            </div>
          </List.Item>
        )}
        locale={{ emptyText: "No matching pages" }}
      />
      <div
        style={{
          padding: "6px 16px",
          borderTop: "1px solid var(--ant-color-border)",
          fontSize: 11,
          color: "var(--ant-color-text-tertiary)",
          display: "flex",
          gap: 12,
        }}
      >
        <span>
          <kbd>↵</kbd> open
        </span>
        <span>
          <kbd>esc</kbd> close
        </span>
      </div>
    </Modal>
  );
}
