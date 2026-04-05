import { CopyableId } from "@/components/common/CopyableId";
import { EmptyState } from "@/components/common/EmptyState";
import { OutcomeBadge } from "@/components/common/OutcomeBadge";
import { RelativeTime } from "@/components/common/RelativeTime";
import { useDecisionQuery, useExplanationQuery } from "@/graphql/generated";
import { useTenantId } from "@/hooks/useTenantId";
import {
  Breadcrumb,
  Card,
  Col,
  Descriptions,
  Row,
  Space,
  Spin,
  Typography,
} from "antd";
import { Link, useParams } from "react-router-dom";
import ExplanationPanel from "./ExplanationPanel";

export default function DecisionDetail() {
  const { id } = useParams<{ id: string }>();
  const tenantId = useTenantId();
  const paymentEventId = id ?? "";

  const { data: dData, loading: dLoading } = useDecisionQuery({
    variables: { tenantId, paymentEventId },
    skip: !paymentEventId,
  });

  const { data: eData, loading: eLoading } = useExplanationQuery({
    variables: { tenantId, paymentEventId },
    skip: !paymentEventId,
  });

  if (dLoading) {
    return (
      <div style={{ textAlign: "center", padding: 64 }}>
        <Spin />
      </div>
    );
  }

  const decision = dData?.decision;
  if (!decision) {
    return <EmptyState title="Decision not found" />;
  }

  return (
    <div>
      <Breadcrumb
        style={{ marginBottom: 16, fontSize: 12 }}
        items={[
          { title: <Link to="/merchant/decisions">Decisions</Link> },
          { title: <CopyableId id={decision.paymentEventId} /> },
        ]}
      />

      <Typography.Title level={4} style={{ marginBottom: 16 }}>
        Decision Detail
      </Typography.Title>

      <Row gutter={[16, 16]}>
        <Col xs={24} lg={12}>
          <Card title="Decision Summary" size="small">
            <Descriptions
              column={1}
              size="small"
              labelStyle={{ width: 140, fontSize: 12, fontWeight: 500 }}
            >
              <Descriptions.Item label="Decision ID">
                <CopyableId id={decision.id} />
              </Descriptions.Item>
              <Descriptions.Item label="Payment Event">
                <CopyableId id={decision.paymentEventId} />
              </Descriptions.Item>
              <Descriptions.Item label="Outcome">
                <OutcomeBadge outcome={decision.outcome} />
              </Descriptions.Item>
              <Descriptions.Item label="Confidence">
                {(decision.confidenceScore * 100).toFixed(1)}%
              </Descriptions.Item>
              <Descriptions.Item label="Reason Codes">
                {decision.reasonCodes.join(", ") || "—"}
              </Descriptions.Item>
              <Descriptions.Item label="Latency">
                {decision.latencyMs}ms
              </Descriptions.Item>
              <Descriptions.Item label="Overridden">
                {decision.overridden ? (
                  <Typography.Text type="warning">Yes</Typography.Text>
                ) : (
                  "—"
                )}
              </Descriptions.Item>
              <Descriptions.Item label="Created">
                <RelativeTime timestamp={decision.createdAt} />
              </Descriptions.Item>
            </Descriptions>
          </Card>

          {/* Cross-portal context link (B3 enhancement) */}
          <Card size="small" style={{ marginTop: 16 }}>
            <Space>
              <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                Related:
              </Typography.Text>
              <Link
                to={`/analyst/cases?paymentEventId=${decision.paymentEventId}`}
              >
                View Cases →
              </Link>
            </Space>
          </Card>
        </Col>

        <Col xs={24} lg={12}>
          <ExplanationPanel
            explanation={eData?.explanation ?? null}
            loading={eLoading}
          />
        </Col>
      </Row>
    </div>
  );
}
