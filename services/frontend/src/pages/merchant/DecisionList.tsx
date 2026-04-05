import { CopyableId } from "@/components/common/CopyableId";
import { OutcomeBadge } from "@/components/common/OutcomeBadge";
import { TableSkeleton } from "@/components/common/PageSkeleton";
import { RelativeTime } from "@/components/common/RelativeTime";
import { useDecisionsQuery, type Decision } from "@/graphql/generated";
import { useTenantId } from "@/hooks/useTenantId";
import { Card, Select, Space, Table, Typography } from "antd";
import type { ColumnsType } from "antd/es/table";
import { useState } from "react";
import { Link } from "react-router-dom";

const PAGE_SIZE = 20;

const columns: ColumnsType<Decision> = [
  {
    title: "Payment",
    dataIndex: "paymentEventId",
    width: 130,
    render: (id: string) => <CopyableId id={id} />,
  },
  {
    title: "Outcome",
    dataIndex: "outcome",
    width: 110,
    render: (v: string) => <OutcomeBadge outcome={v} />,
  },
  {
    title: "Confidence",
    dataIndex: "confidenceScore",
    width: 100,
    align: "right",
    render: (v: number) => `${(v * 100).toFixed(1)}%`,
  },
  {
    title: "Reason Codes",
    dataIndex: "reasonCodes",
    ellipsis: true,
    render: (codes: string[]) => codes.join(", ") || "—",
  },
  {
    title: "Latency",
    dataIndex: "latencyMs",
    width: 90,
    align: "right",
    render: (v: number) => `${v}ms`,
  },
  {
    title: "Overridden",
    dataIndex: "overridden",
    width: 90,
    align: "center",
    render: (v: boolean) => (v ? "Yes" : "—"),
  },
  {
    title: "Time",
    dataIndex: "createdAt",
    width: 140,
    render: (v: string) => <RelativeTime timestamp={v} />,
  },
  {
    title: "",
    key: "action",
    width: 60,
    render: (_, record) => (
      <Link to={`/merchant/decisions/${record.paymentEventId}`}>View</Link>
    ),
  },
];

export default function DecisionList() {
  const tenantId = useTenantId();
  const [page, setPage] = useState(1);
  const [outcomeFilter, setOutcomeFilter] = useState<string | undefined>();

  const { data, loading } = useDecisionsQuery({
    variables: {
      tenantId,
      page,
      pageSize: PAGE_SIZE,
      outcomeFilter,
    },
  });

  if (loading && !data) return <TableSkeleton rows={10} />;

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
          Decisions
        </Typography.Title>
        <Space>
          <Select
            placeholder="Filter outcome"
            allowClear
            style={{ width: 160 }}
            value={outcomeFilter}
            onChange={(v) => {
              setOutcomeFilter(v);
              setPage(1);
            }}
            options={[
              { value: "APPROVE", label: "Approve" },
              { value: "FLAG", label: "Flag" },
              { value: "REVIEW", label: "Review" },
              { value: "BLOCK", label: "Block" },
            ]}
          />
        </Space>
      </div>

      <Card bodyStyle={{ padding: 0 }}>
        <Table
          rowKey="id"
          columns={columns}
          dataSource={data?.decisions.decisions ?? []}
          loading={loading}
          size="small"
          pagination={{
            current: page,
            pageSize: PAGE_SIZE,
            total: data?.decisions.total ?? 0,
            onChange: setPage,
            showSizeChanger: false,
            showTotal: (total) => (
              <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                {total} decisions
              </Typography.Text>
            ),
          }}
        />
      </Card>
    </div>
  );
}
