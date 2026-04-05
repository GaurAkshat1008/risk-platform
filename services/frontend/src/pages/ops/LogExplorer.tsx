import { CopyableId } from "@/components/common/CopyableId";
import { TableSkeleton } from "@/components/common/PageSkeleton";
import { RelativeTime } from "@/components/common/RelativeTime";
import { SeverityTag } from "@/components/common/SeverityTag";
import { useQueryLogsQuery, type LogEntry } from "@/graphql/generated";
import { usePolling } from "@/hooks/usePolling";
import { ReloadOutlined } from "@ant-design/icons";
import {
  Button,
  Card,
  Input,
  Select,
  Space,
  Table,
  Typography,
} from "antd";
import type { ColumnsType } from "antd/es/table";
import { useState } from "react";
import { Link } from "react-router-dom";

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

const SEVERITIES = ["DEBUG", "INFO", "WARN", "ERROR", "FATAL"];

const columns: ColumnsType<LogEntry> = [
  {
    title: "Time",
    dataIndex: "timestamp",
    width: 140,
    render: (v: string) => <RelativeTime timestamp={v} />,
  },
  {
    title: "Severity",
    dataIndex: "severity",
    width: 80,
    render: (v: string) => <SeverityTag severity={v} />,
  },
  {
    title: "Service",
    dataIndex: "service",
    width: 130,
    render: (v: string) => (
      <Typography.Text code style={{ fontSize: 11 }}>
        {v}
      </Typography.Text>
    ),
  },
  {
    title: "Message",
    dataIndex: "message",
    ellipsis: true,
  },
  {
    title: "Trace",
    dataIndex: "traceId",
    width: 120,
    render: (v: string) =>
      v ? (
        <Link to={`/ops/traces?traceId=${v}`}>
          <CopyableId id={v} />
        </Link>
      ) : (
        "—"
      ),
  },
];

export default function LogExplorer() {
  const [service, setService] = useState<string | undefined>();
  const [severity, setSeverity] = useState<string | undefined>();
  const [search, setSearch] = useState("");
  const [pageToken, setPageToken] = useState<string | undefined>();

  const { data, loading, refetch } = useQueryLogsQuery({
    variables: {
      query: {
        service,
        severity,
        messageContains: search || undefined,
        pageSize: 50,
        pageToken,
      },
    },
  });

  usePolling(() => refetch(), 15000);

  if (loading && !data) return <TableSkeleton rows={15} />;

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
          Log Explorer
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
            options={SEVERITIES.map((s) => ({ value: s, label: s }))}
          />
          <Input.Search
            placeholder="Search messages…"
            allowClear
            style={{ width: 260 }}
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            onSearch={() => refetch()}
          />
        </Space>
      </Card>

      <Card bodyStyle={{ padding: 0 }}>
        <Table
          rowKey="id"
          columns={columns}
          dataSource={data?.queryLogs.entries ?? []}
          loading={loading}
          size="small"
          pagination={false}
          rowClassName={(record) =>
            record.severity === "ERROR" || record.severity === "FATAL"
              ? "log-row-error"
              : ""
          }
        />
        {data?.queryLogs.nextPageToken && (
          <div style={{ textAlign: "center", padding: 12 }}>
            <Typography.Link
              onClick={() => setPageToken(data.queryLogs.nextPageToken)}
            >
              Load more
            </Typography.Link>
          </div>
        )}
      </Card>
    </div>
  );
}
