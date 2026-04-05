import { TableSkeleton } from "@/components/common/PageSkeleton";
import { RelativeTime } from "@/components/common/RelativeTime";
import { useAlertsQuery, type Alert } from "@/graphql/generated";
import { usePolling } from "@/hooks/usePolling";
import { BellOutlined, ReloadOutlined } from "@ant-design/icons";
import { Button, Card, Select, Space, Table, Tag, Typography } from "antd";
import type { ColumnsType } from "antd/es/table";
import { useState } from "react";

const SERVICES = [
  "ingestion",
  "decision",
  "rules-engine",
  "case-management",
  "workflow",
  "identity-access",
  "tenant-config",
  "notification",
  "audit-trail",
  "explanation",
  "graphql-bff",
];

const severityColor: Record<string, string> = {
  critical: "red",
  warning: "orange",
  info: "blue",
};

const stateColor: Record<string, string> = {
  firing: "red",
  pending: "orange",
  resolved: "green",
};

const columns: ColumnsType<Alert> = [
  {
    title: "Alert",
    dataIndex: "name",
    ellipsis: true,
    render: (v: string) => (
      <Typography.Text strong style={{ fontSize: 13 }}>
        {v}
      </Typography.Text>
    ),
  },
  {
    title: "Service",
    dataIndex: "service",
    width: 140,
    render: (v: string) => (
      <Typography.Text code style={{ fontSize: 11 }}>
        {v}
      </Typography.Text>
    ),
  },
  {
    title: "Severity",
    dataIndex: "severity",
    width: 90,
    render: (v: string) => (
      <Tag color={severityColor[v] ?? "default"} style={{ fontSize: 11 }}>
        {v.toUpperCase()}
      </Tag>
    ),
  },
  {
    title: "State",
    dataIndex: "state",
    width: 90,
    render: (v: string) => (
      <Tag
        color={stateColor[v] ?? "default"}
        style={{ fontSize: 11 }}
        className={v === "firing" ? "pulse-live" : undefined}
      >
        {v.toUpperCase()}
      </Tag>
    ),
  },
  {
    title: "Summary",
    dataIndex: "summary",
    ellipsis: true,
  },
  {
    title: "Fired At",
    dataIndex: "firedAt",
    width: 140,
    render: (v: string) => <RelativeTime timestamp={v} />,
  },
];

export default function AlertCenter() {
  const [service, setService] = useState<string | undefined>();
  const [severity, setSeverity] = useState<string | undefined>();
  const [activeOnly] = useState(true);

  const { data, loading, refetch } = useAlertsQuery({
    variables: {
      service,
      severity,
      activeOnly,
      limit: 100,
    },
  });

  usePolling(() => refetch(), 10000);

  if (loading && !data) return <TableSkeleton rows={8} />;

  const alerts = data?.alerts ?? [];
  const firingCount = alerts.filter((a) => a.state === "firing").length;

  return (
    <div>
      <div
        style={{
          display: "flex",
          alignItems: "center",
          justifyContent: "space-between",
          marginBottom: 16,
        }}
      >
        <Space>
          <Typography.Title level={4} style={{ margin: 0 }}>
            Alert Center
          </Typography.Title>
          {firingCount > 0 && (
            <Tag color="red" icon={<BellOutlined />} className="pulse-live">
              {firingCount} firing
            </Tag>
          )}
        </Space>
        <Space>
          <Select
            placeholder="Service"
            allowClear
            style={{ width: 160 }}
            value={service}
            onChange={setService}
            options={SERVICES.map((s) => ({ value: s, label: s }))}
          />
          <Select
            placeholder="Severity"
            allowClear
            style={{ width: 120 }}
            value={severity}
            onChange={setSeverity}
            options={[
              { value: "critical", label: "Critical" },
              { value: "warning", label: "Warning" },
              { value: "info", label: "Info" },
            ]}
          />
          <Button
            icon={<ReloadOutlined />}
            onClick={() => refetch()}
            size="small"
          >
            Refresh
          </Button>
        </Space>
      </div>

      <Card bodyStyle={{ padding: 0 }}>
        <Table
          rowKey={(r) => `${r.name}-${r.service}-${r.firedAt}`}
          columns={columns}
          dataSource={alerts}
          loading={loading}
          size="small"
          pagination={{ pageSize: 25, showSizeChanger: false }}
        />
      </Card>
    </div>
  );
}
