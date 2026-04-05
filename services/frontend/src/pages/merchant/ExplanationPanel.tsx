import { EmptyState } from "@/components/common/EmptyState";
import { OutcomeBadge } from "@/components/common/OutcomeBadge";
import type {
  Explanation,
  FeatureValue,
  RuleContribution,
} from "@/graphql/generated";
import {
  Card,
  Collapse,
  Descriptions,
  Spin,
  Table,
  Tag,
  Typography,
} from "antd";
import type { ColumnsType } from "antd/es/table";

interface Props {
  explanation: Explanation | null;
  loading: boolean;
}

const ruleColumns: ColumnsType<RuleContribution> = [
  {
    title: "Rule",
    dataIndex: "ruleName",
    ellipsis: true,
  },
  {
    title: "Matched",
    dataIndex: "matched",
    width: 80,
    align: "center",
    render: (v: boolean) =>
      v ? (
        <Tag color="red" style={{ fontSize: 11 }}>
          HIT
        </Tag>
      ) : (
        <Tag style={{ fontSize: 11 }}>MISS</Tag>
      ),
  },
  {
    title: "Action",
    dataIndex: "action",
    width: 80,
    render: (v: string) => <OutcomeBadge outcome={v} />,
  },
  {
    title: "Reason",
    dataIndex: "reason",
    ellipsis: true,
  },
];

const featureColumns: ColumnsType<FeatureValue> = [
  { title: "Feature", dataIndex: "name", ellipsis: true },
  {
    title: "Value",
    dataIndex: "value",
    width: 160,
    render: (v: string) => <Typography.Text code>{v}</Typography.Text>,
  },
];

export default function ExplanationPanel({ explanation, loading }: Props) {
  if (loading) {
    return (
      <Card title="Explanation" size="small">
        <div style={{ textAlign: "center", padding: 32 }}>
          <Spin />
        </div>
      </Card>
    );
  }

  if (!explanation) {
    return (
      <Card title="Explanation" size="small">
        <EmptyState title="No explanation available" />
      </Card>
    );
  }

  return (
    <Card title="Explanation" size="small">
      <Descriptions
        column={1}
        size="small"
        labelStyle={{ width: 130, fontSize: 12, fontWeight: 500 }}
        style={{ marginBottom: 16 }}
      >
        <Descriptions.Item label="Outcome">
          <OutcomeBadge outcome={explanation.outcome} />
        </Descriptions.Item>
        <Descriptions.Item label="Confidence">
          {(explanation.confidenceScore * 100).toFixed(1)}%
        </Descriptions.Item>
        <Descriptions.Item label="Policy Version">
          <Typography.Text code>{explanation.policyVersion}</Typography.Text>
        </Descriptions.Item>
      </Descriptions>

      {explanation.narrative && (
        <Typography.Paragraph
          type="secondary"
          style={{
            fontSize: 12,
            padding: "8px 12px",
            background: "var(--ant-color-fill-quaternary)",
            borderRadius: 4,
            marginBottom: 16,
          }}
        >
          {explanation.narrative}
        </Typography.Paragraph>
      )}

      <Collapse
        ghost
        size="small"
        items={[
          {
            key: "rules",
            label: `Rule Contributions (${explanation.ruleContributions.length})`,
            children: (
              <Table
                rowKey="ruleId"
                columns={ruleColumns}
                dataSource={explanation.ruleContributions}
                pagination={false}
                size="small"
              />
            ),
          },
          {
            key: "features",
            label: `Feature Values (${explanation.featureValues.length})`,
            children: (
              <Table
                rowKey="name"
                columns={featureColumns}
                dataSource={explanation.featureValues}
                pagination={false}
                size="small"
              />
            ),
          },
        ]}
      />
    </Card>
  );
}
