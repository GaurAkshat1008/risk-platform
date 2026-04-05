import { CopyableId } from "@/components/common/CopyableId";
import { OutcomeBadge } from "@/components/common/OutcomeBadge";
import { TableSkeleton } from "@/components/common/PageSkeleton";
import { SlaCountdown } from "@/components/common/SlaCountdown";
import { StatusTag } from "@/components/common/StatusTag";
import {
  useCasesQuery,
  type Case,
  type CasePriority,
  type CaseStatus,
} from "@/graphql/generated";
import { useAuth } from "@/hooks/useAuth";
import { useTenantId } from "@/hooks/useTenantId";
import { Card, Select, Space, Table, Tag, Typography } from "antd";
import type { ColumnsType } from "antd/es/table";
import { useState } from "react";
import { Link } from "react-router-dom";

const priorityColor: Record<string, string> = {
  CRITICAL: "red",
  HIGH: "orange",
  MEDIUM: "blue",
  LOW: "default",
};

const PAGE_SIZE = 20;

const columns: ColumnsType<Case> = [
  {
    title: "Case",
    dataIndex: "id",
    width: 120,
    render: (id: string) => <CopyableId id={id} />,
  },
  {
    title: "Priority",
    dataIndex: "priority",
    width: 90,
    render: (v: CasePriority) => (
      <Tag color={priorityColor[v] ?? "default"} style={{ fontSize: 11 }}>
        {v}
      </Tag>
    ),
    sorter: (a, b) => {
      const order = ["CRITICAL", "HIGH", "MEDIUM", "LOW"];
      return order.indexOf(a.priority) - order.indexOf(b.priority);
    },
  },
  {
    title: "Status",
    dataIndex: "status",
    width: 110,
    render: (v: string) => <StatusTag status={v} />,
  },
  {
    title: "Outcome",
    dataIndex: "outcome",
    width: 100,
    render: (v: string) => <OutcomeBadge outcome={v} />,
  },
  {
    title: "SLA",
    dataIndex: "slaDeadline",
    width: 120,
    render: (v: string) => <SlaCountdown deadline={v} />,
  },
  {
    title: "Assignee",
    dataIndex: "assigneeId",
    width: 100,
    ellipsis: true,
    render: (v: string) =>
      v || <Typography.Text type="secondary">Unassigned</Typography.Text>,
  },
  {
    title: "Payment",
    dataIndex: "paymentEventId",
    width: 120,
    render: (id: string) => (
      <Link to={`/merchant/decisions/${id}`}>
        <CopyableId id={id} />
      </Link>
    ),
  },
  {
    title: "",
    key: "action",
    width: 60,
    render: (_, record) => <Link to={`/analyst/cases/${record.id}`}>Open</Link>,
  },
];

export default function CaseQueue() {
  const tenantId = useTenantId();
  const { user } = useAuth();
  const [statusFilter, setStatusFilter] = useState<CaseStatus | undefined>();
  const [assigneeFilter, setAssigneeFilter] = useState<string | undefined>();
  const [pageToken, setPageToken] = useState<string | undefined>();

  const { data, loading } = useCasesQuery({
    variables: {
      tenantId,
      status: statusFilter,
      assigneeId: assigneeFilter,
      pageSize: PAGE_SIZE,
      pageToken,
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
          Case Queue
        </Typography.Title>
        <Space>
          <Select
            placeholder="Status"
            allowClear
            style={{ width: 140 }}
            value={statusFilter}
            onChange={(v) => {
              setStatusFilter(v);
              setPageToken(undefined);
            }}
            options={[
              { value: "OPEN", label: "Open" },
              { value: "IN_REVIEW", label: "In Review" },
              { value: "ESCALATED", label: "Escalated" },
              { value: "RESOLVED", label: "Resolved" },
            ]}
          />
          <Select
            placeholder="Assignee"
            allowClear
            style={{ width: 140 }}
            value={assigneeFilter}
            onChange={(v) => {
              setAssigneeFilter(v);
              setPageToken(undefined);
            }}
            options={[{ value: user.userId, label: "My Cases" }]}
          />
        </Space>
      </div>

      <Card bodyStyle={{ padding: 0 }}>
        <Table
          rowKey="id"
          columns={columns}
          dataSource={data?.cases.cases ?? []}
          loading={loading}
          size="small"
          pagination={false}
        />
        {data?.cases.nextPageToken && (
          <div style={{ textAlign: "center", padding: 12 }}>
            <Typography.Link
              onClick={() => setPageToken(data.cases.nextPageToken)}
            >
              Load more
            </Typography.Link>
          </div>
        )}
      </Card>
    </div>
  );
}
