import { CopyableId } from "@/components/common/CopyableId";
import { TableSkeleton } from "@/components/common/PageSkeleton";
import { RelativeTime } from "@/components/common/RelativeTime";
import { useQueryTracesQuery, type TraceSpan } from "@/graphql/generated";
import { ReloadOutlined } from "@ant-design/icons";
import {
  Button,
  Card,
  Input,
  Select,
  Space,
  Table,
  Tag,
  Typography,
} from "antd";
import type { ColumnsType } from "antd/es/table";
import { useState } from "react";
import { useSearchParams } from "react-router-dom";

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

const statusColor: Record<string, string> = {
  OK: "green",
  ERROR: "red",
  UNSET: "default",
};

const columns: ColumnsType<TraceSpan> = [
  {
    title: "Trace ID",
    dataIndex: "traceId",
    width: 130,
    render: (v: string) => <CopyableId id={v} />,
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
    title: "Operation",
    dataIndex: "operation",
    ellipsis: true,
  },
  {
    title: "Duration",
    dataIndex: "durationMs",
    width: 100,
    align: "right",
    render: (v: number) => (
      <span
        style={{
          fontVariantNumeric: "tabular-nums",
          color: v > 500 ? "#dc2626" : v > 200 ? "#d97706" : undefined,
        }}
      >
        {v}ms
      </span>
    ),
    sorter: (a, b) => a.durationMs - b.durationMs,
  },
  {
    title: "Status",
    dataIndex: "status",
    width: 80,
    render: (v: string) => <Tag color={statusColor[v] ?? "default"}>{v}</Tag>,
  },
  {
    title: "Started",
    dataIndex: "startTime",
    width: 140,
    render: (v: string) => <RelativeTime timestamp={v} />,
  },
];

export default function TraceViewer() {
  const [searchParams] = useSearchParams();
  const [traceId, setTraceId] = useState(searchParams.get("traceId") ?? "");
  const [service, setService] = useState<string | undefined>();

  const { data, loading, refetch } = useQueryTracesQuery({
    variables: {
      traceId: traceId || undefined,
      service,
      limit: 100,
    },
  });

  if (loading && !data) return <TableSkeleton rows={12} />;

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
        <Typography.Title level={4} style={{ margin: 0 }}>
          Trace Viewer
        </Typography.Title>
        <Button
          icon={<ReloadOutlined />}
          onClick={() => refetch()}
          size="small"
        >
          Refresh
        </Button>
      </div>

      <Card size="small" style={{ marginBottom: 16 }}>
        <Space wrap>
          <Input.Search
            placeholder="Trace ID"
            allowClear
            style={{ width: 280 }}
            value={traceId}
            onChange={(e) => setTraceId(e.target.value)}
            onSearch={() => refetch()}
          />
          <Select
            placeholder="Service"
            allowClear
            style={{ width: 160 }}
            value={service}
            onChange={setService}
            options={SERVICES.map((s) => ({ value: s, label: s }))}
          />
        </Space>
      </Card>

      <Card bodyStyle={{ padding: 0 }}>
        <Table
          rowKey={(r) => `${r.traceId}-${r.spanId}`}
          columns={columns}
          dataSource={data?.queryTraces ?? []}
          loading={loading}
          size="small"
          pagination={{ pageSize: 50, showSizeChanger: false }}
        />
      </Card>
    </div>
  );
}
