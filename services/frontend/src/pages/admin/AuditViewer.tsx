import { CopyableId } from "@/components/common/CopyableId";
import { TableSkeleton } from "@/components/common/PageSkeleton";
import { RelativeTime } from "@/components/common/RelativeTime";
import {
  useAuditTrailQuery,
  useVerifyChainIntegrityQuery,
  type AuditEvent,
} from "@/graphql/generated";
import { useTenantId } from "@/hooks/useTenantId";
import { ReloadOutlined, SafetyCertificateOutlined } from "@ant-design/icons";
import { Button, Card, Select, Space, Table, Tag, Typography } from "antd";
import type { ColumnsType } from "antd/es/table";
import { useState } from "react";

const columns: ColumnsType<AuditEvent> = [
  {
    title: "Time",
    dataIndex: "occurredAt",
    width: 140,
    render: (v: string) => <RelativeTime timestamp={v} />,
  },
  {
    title: "Actor",
    dataIndex: "actorId",
    width: 120,
    render: (v: string) => <CopyableId id={v} />,
  },
  {
    title: "Action",
    dataIndex: "action",
    width: 120,
    render: (v: string) => <Tag style={{ fontSize: 11 }}>{v}</Tag>,
  },
  {
    title: "Resource Type",
    dataIndex: "resourceType",
    width: 130,
    render: (v: string) => (
      <Typography.Text code style={{ fontSize: 11 }}>
        {v}
      </Typography.Text>
    ),
  },
  {
    title: "Resource ID",
    dataIndex: "resourceId",
    width: 120,
    render: (v: string) => <CopyableId id={v} />,
  },
  {
    title: "Topic",
    dataIndex: "sourceTopic",
    width: 140,
    render: (v: string) => (
      <Typography.Text code style={{ fontSize: 11 }}>
        {v}
      </Typography.Text>
    ),
  },
  {
    title: "Hash",
    dataIndex: "hash",
    width: 100,
    render: (v: string) => (
      <Typography.Text code style={{ fontSize: 10 }}>
        {v?.slice(0, 12)}…
      </Typography.Text>
    ),
  },
];

export default function AuditViewer() {
  const tenantId = useTenantId();
  const [action, setAction] = useState<string | undefined>();
  const [resourceType, setResourceType] = useState<string | undefined>();
  const [pageToken, setPageToken] = useState<string | undefined>();

  const { data, loading, refetch } = useAuditTrailQuery({
    variables: {
      query: {
        tenantId,
        action,
        resourceType,
        pageSize: 30,
        pageToken,
      },
    },
  });

  const { data: chainData, refetch: recheckChain } =
    useVerifyChainIntegrityQuery({
      variables: { tenantId, limit: 500 },
    });

  if (loading && !data) return <TableSkeleton rows={12} />;

  const chain = chainData?.verifyChainIntegrity;

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
          Audit Trail
        </Typography.Title>
        <Space>
          {chain && (
            <Tag
              color={chain.valid ? "green" : "red"}
              icon={<SafetyCertificateOutlined />}
              style={{ cursor: "pointer" }}
              onClick={() => recheckChain()}
            >
              {chain.valid ? "Chain valid" : `Broken at ${chain.brokenAtId}`} (
              {chain.eventsChecked} checked)
            </Tag>
          )}
          <Button
            icon={<ReloadOutlined />}
            onClick={() => refetch()}
            size="small"
          >
            Refresh
          </Button>
        </Space>
      </div>

      <Card size="small" style={{ marginBottom: 16 }}>
        <Space wrap>
          <Select
            placeholder="Action"
            allowClear
            style={{ width: 160 }}
            value={action}
            onChange={setAction}
            options={["CREATE", "UPDATE", "DELETE", "LOGIN", "OVERRIDE"].map(
              (a) => ({ value: a, label: a }),
            )}
          />
          <Select
            placeholder="Resource type"
            allowClear
            style={{ width: 160 }}
            value={resourceType}
            onChange={setResourceType}
            options={[
              "DECISION",
              "CASE",
              "RULE",
              "WORKFLOW",
              "TENANT",
              "NOTIFICATION",
            ].map((t) => ({ value: t, label: t }))}
          />
        </Space>
      </Card>

      <Card bodyStyle={{ padding: 0 }}>
        <Table
          rowKey="id"
          columns={columns}
          dataSource={data?.auditTrail.events ?? []}
          loading={loading}
          size="small"
          pagination={false}
        />
        {data?.auditTrail.nextPageToken && (
          <div style={{ textAlign: "center", padding: 12 }}>
            <Typography.Link
              onClick={() => setPageToken(data.auditTrail.nextPageToken)}
            >
              Load more
            </Typography.Link>
          </div>
        )}
      </Card>
    </div>
  );
}
