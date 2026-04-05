import { CopyableId } from "@/components/common/CopyableId";
import { EmptyState } from "@/components/common/EmptyState";
import { OutcomeBadge } from "@/components/common/OutcomeBadge";
import { RelativeTime } from "@/components/common/RelativeTime";
import { SlaCountdown } from "@/components/common/SlaCountdown";
import { StatusTag } from "@/components/common/StatusTag";
import {
  useAssignCaseMutation,
  useCaseDetailQuery,
  useEscalateCaseMutation,
  useOverrideDecisionMutation,
  useUpdateCaseStatusMutation,
  type CaseStatus,
  type Outcome,
} from "@/graphql/generated";
import { useAuth } from "@/hooks/useAuth";
import { showGqlError } from "@/utils/apolloErrors";
import {
  CheckCircleOutlined,
  ExclamationCircleOutlined,
  SwapOutlined,
  UserSwitchOutlined,
} from "@ant-design/icons";
import {
  Breadcrumb,
  Button,
  Card,
  Col,
  Descriptions,
  Input,
  Modal,
  Row,
  Select,
  Space,
  Spin,
  Tag,
  Typography,
  message,
} from "antd";
import { useState } from "react";
import { Link, useParams } from "react-router-dom";

export default function CaseDetail() {
  const { id: caseId } = useParams<{ id: string }>();
  const { user } = useAuth();

  const { data, loading, refetch } = useCaseDetailQuery({
    variables: { caseId: caseId! },
    skip: !caseId,
  });

  /* ── Mutations ── */
  const [updateStatus, { loading: updLoading }] = useUpdateCaseStatusMutation({
    onCompleted: () => {
      message.success("Status updated");
      refetch();
    },
    onError: showGqlError,
  });

  const [escalate, { loading: escLoading }] = useEscalateCaseMutation({
    onCompleted: () => {
      message.success("Case escalated");
      refetch();
    },
    onError: showGqlError,
  });

  const [assign, { loading: assLoading }] = useAssignCaseMutation({
    onCompleted: () => {
      message.success("Case assigned");
      refetch();
    },
    onError: showGqlError,
  });

  const [overrideDecision, { loading: ovrLoading }] =
    useOverrideDecisionMutation({
      onCompleted: () => {
        message.success("Decision overridden");
        refetch();
      },
      onError: showGqlError,
    });

  /* ── Override modal state ── */
  const [overrideOpen, setOverrideOpen] = useState(false);
  const [newOutcome, setNewOutcome] = useState<Outcome>("APPROVE");
  const [overrideReason, setOverrideReason] = useState("");

  /* ── Escalate modal state ── */
  const [escalateOpen, setEscalateOpen] = useState(false);
  const [escalateReason, setEscalateReason] = useState("");

  /* ── Status update modal state ── */
  const [statusOpen, setStatusOpen] = useState(false);
  const [newStatus, setNewStatus] = useState<CaseStatus>("RESOLVED");
  const [statusNotes, setStatusNotes] = useState("");

  if (loading) {
    return (
      <div style={{ textAlign: "center", padding: 64 }}>
        <Spin />
      </div>
    );
  }

  const c = data?.case;
  if (!c) return <EmptyState title="Case not found" />;

  const isResolved = c.status === "RESOLVED";

  return (
    <div>
      <Breadcrumb
        style={{ marginBottom: 16, fontSize: 12 }}
        items={[
          { title: <Link to="/analyst/cases">Case Queue</Link> },
          { title: <CopyableId id={c.id} /> },
        ]}
      />

      <div
        style={{
          display: "flex",
          alignItems: "center",
          justifyContent: "space-between",
          marginBottom: 16,
        }}
      >
        <Typography.Title level={4} style={{ margin: 0 }}>
          Case Detail
        </Typography.Title>
        {!isResolved && (
          <Space>
            <Button
              size="small"
              icon={<UserSwitchOutlined />}
              loading={assLoading}
              onClick={() =>
                assign({
                  variables: {
                    input: {
                      caseId: c.id,
                      assigneeId: user.userId,
                      actorId: user.userId,
                    },
                  },
                  /* A2: optimistic UI */
                  optimisticResponse: {
                    assignCase: {
                      ...c,
                      assigneeId: user.userId,
                      updatedAt: new Date().toISOString(),
                    },
                  },
                })
              }
            >
              Assign to me
            </Button>
            <Button
              size="small"
              icon={<CheckCircleOutlined />}
              onClick={() => setStatusOpen(true)}
            >
              Update Status
            </Button>
            <Button
              size="small"
              icon={<SwapOutlined />}
              onClick={() => setOverrideOpen(true)}
            >
              Override
            </Button>
            <Button
              size="small"
              danger
              icon={<ExclamationCircleOutlined />}
              onClick={() => setEscalateOpen(true)}
            >
              Escalate
            </Button>
          </Space>
        )}
      </div>

      <Row gutter={[16, 16]}>
        <Col xs={24} lg={14}>
          <Card title="Case Info" size="small">
            <Descriptions
              column={1}
              size="small"
              labelStyle={{ width: 140, fontSize: 12, fontWeight: 500 }}
            >
              <Descriptions.Item label="Case ID">
                <CopyableId id={c.id} />
              </Descriptions.Item>
              <Descriptions.Item label="Decision ID">
                <CopyableId id={c.decisionId} />
              </Descriptions.Item>
              <Descriptions.Item label="Payment Event">
                <Link to={`/merchant/decisions/${c.paymentEventId}`}>
                  <CopyableId id={c.paymentEventId} />
                </Link>
              </Descriptions.Item>
              <Descriptions.Item label="Status">
                <StatusTag status={c.status} />
              </Descriptions.Item>
              <Descriptions.Item label="Priority">
                <Tag
                  color={
                    c.priority === "CRITICAL"
                      ? "red"
                      : c.priority === "HIGH"
                        ? "orange"
                        : "blue"
                  }
                >
                  {c.priority}
                </Tag>
              </Descriptions.Item>
              <Descriptions.Item label="Outcome">
                <OutcomeBadge outcome={c.outcome} />
              </Descriptions.Item>
              <Descriptions.Item label="Assignee">
                {c.assigneeId || "Unassigned"}
              </Descriptions.Item>
              <Descriptions.Item label="Created">
                <RelativeTime timestamp={c.createdAt} />
              </Descriptions.Item>
              <Descriptions.Item label="Updated">
                <RelativeTime timestamp={c.updatedAt} />
              </Descriptions.Item>
            </Descriptions>
          </Card>
        </Col>

        <Col xs={24} lg={10}>
          <Card title="SLA" size="small">
            <div style={{ textAlign: "center", padding: 24 }}>
              <SlaCountdown deadline={c.slaDeadline} />
              <Typography.Text
                type="secondary"
                style={{ display: "block", marginTop: 8, fontSize: 12 }}
              >
                Deadline: {new Date(c.slaDeadline).toLocaleString()}
              </Typography.Text>
            </div>
          </Card>
        </Col>
      </Row>

      {/* ── Override Decision Modal ── */}
      <Modal
        title="Override Decision"
        open={overrideOpen}
        onCancel={() => setOverrideOpen(false)}
        confirmLoading={ovrLoading}
        onOk={() => {
          overrideDecision({
            variables: {
              input: {
                decisionId: c.decisionId,
                analystId: user.userId,
                newOutcome: newOutcome,
                reason: overrideReason,
              },
            },
          });
          setOverrideOpen(false);
        }}
        okButtonProps={{ disabled: !overrideReason.trim() }}
      >
        <Space direction="vertical" style={{ width: "100%" }} size="middle">
          <div>
            <Typography.Text
              strong
              style={{ display: "block", marginBottom: 4 }}
            >
              New Outcome
            </Typography.Text>
            <Select
              style={{ width: "100%" }}
              value={newOutcome}
              onChange={setNewOutcome}
              options={[
                { value: "APPROVE", label: "Approve" },
                { value: "FLAG", label: "Flag" },
                { value: "REVIEW", label: "Review" },
                { value: "BLOCK", label: "Block" },
              ]}
            />
          </div>
          <div>
            <Typography.Text
              strong
              style={{ display: "block", marginBottom: 4 }}
            >
              Reason
            </Typography.Text>
            <Input.TextArea
              rows={3}
              value={overrideReason}
              onChange={(e) => setOverrideReason(e.target.value)}
              placeholder="Explain why you're overriding this decision…"
            />
          </div>
        </Space>
      </Modal>

      {/* ── Update Status Modal ── */}
      <Modal
        title="Update Case Status"
        open={statusOpen}
        onCancel={() => setStatusOpen(false)}
        confirmLoading={updLoading}
        onOk={() => {
          updateStatus({
            variables: {
              input: {
                caseId: c.id,
                status: newStatus,
                actorId: user.userId,
                notes: statusNotes,
              },
            },
            optimisticResponse: {
              updateCaseStatus: {
                ...c,
                status: newStatus,
                updatedAt: new Date().toISOString(),
              },
            },
          });
          setStatusOpen(false);
        }}
      >
        <Space direction="vertical" style={{ width: "100%" }} size="middle">
          <div>
            <Typography.Text
              strong
              style={{ display: "block", marginBottom: 4 }}
            >
              New Status
            </Typography.Text>
            <Select
              style={{ width: "100%" }}
              value={newStatus}
              onChange={setNewStatus}
              options={[
                { value: "IN_REVIEW", label: "In Review" },
                { value: "RESOLVED", label: "Resolved" },
              ]}
            />
          </div>
          <div>
            <Typography.Text
              strong
              style={{ display: "block", marginBottom: 4 }}
            >
              Notes
            </Typography.Text>
            <Input.TextArea
              rows={2}
              value={statusNotes}
              onChange={(e) => setStatusNotes(e.target.value)}
              placeholder="Optional notes…"
            />
          </div>
        </Space>
      </Modal>

      {/* ── Escalate Modal ── */}
      <Modal
        title="Escalate Case"
        open={escalateOpen}
        onCancel={() => setEscalateOpen(false)}
        confirmLoading={escLoading}
        okType="danger"
        onOk={() => {
          escalate({
            variables: {
              input: {
                caseId: c.id,
                actorId: user.userId,
                reason: escalateReason,
              },
            },
          });
          setEscalateOpen(false);
        }}
        okButtonProps={{ disabled: !escalateReason.trim() }}
      >
        <div>
          <Typography.Text strong style={{ display: "block", marginBottom: 4 }}>
            Reason for escalation
          </Typography.Text>
          <Input.TextArea
            rows={3}
            value={escalateReason}
            onChange={(e) => setEscalateReason(e.target.value)}
            placeholder="Explain why this case needs escalation…"
          />
        </div>
      </Modal>
    </div>
  );
}
