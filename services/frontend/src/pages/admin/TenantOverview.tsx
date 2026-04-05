import { CopyableId } from "@/components/common/CopyableId";
import { CardSkeleton } from "@/components/common/PageSkeleton";
import { RelativeTime } from "@/components/common/RelativeTime";
import { StatusTag } from "@/components/common/StatusTag";
import { useTenantConfigQuery, type FeatureFlag } from "@/graphql/generated";
import { useTenantId } from "@/hooks/useTenantId";
import {
  Card,
  Col,
  Descriptions,
  Progress,
  Row,
  Switch,
  Table,
  Tag,
  Typography,
} from "antd";
import type { ColumnsType } from "antd/es/table";

const flagColumns: ColumnsType<FeatureFlag> = [
  {
    title: "Flag",
    dataIndex: "key",
    render: (v: string) => <Typography.Text code>{v}</Typography.Text>,
  },
  {
    title: "Enabled",
    dataIndex: "enabled",
    width: 80,
    align: "center",
    render: (v: boolean) => <Switch checked={v} size="small" disabled />,
  },
  {
    title: "Rollout",
    dataIndex: "rolloutPercentage",
    width: 140,
    render: (v: number) => (
      <Progress percent={v} size="small" style={{ width: 100 }} />
    ),
  },
];

export default function TenantOverview() {
  const tenantId = useTenantId();

  const { data, loading } = useTenantConfigQuery({
    variables: { tenantId },
  });

  if (loading && !data) return <CardSkeleton />;

  const tenant = data?.tenantConfig;
  if (!tenant)
    return <Typography.Text type="secondary">Tenant not found</Typography.Text>;

  return (
    <div>
      <Typography.Title level={4} style={{ marginBottom: 16 }}>
        Tenant Configuration
      </Typography.Title>

      <Row gutter={[16, 16]}>
        <Col xs={24} lg={12}>
          <Card title="Tenant Details" size="small">
            <Descriptions
              column={1}
              size="small"
              labelStyle={{ width: 160, fontSize: 12, fontWeight: 500 }}
            >
              <Descriptions.Item label="Tenant ID">
                <CopyableId id={tenant.id} />
              </Descriptions.Item>
              <Descriptions.Item label="Name">{tenant.name}</Descriptions.Item>
              <Descriptions.Item label="Status">
                <StatusTag status={tenant.status} />
              </Descriptions.Item>
              <Descriptions.Item label="Created">
                <RelativeTime timestamp={tenant.createdAt} />
              </Descriptions.Item>
              <Descriptions.Item label="Config Version">
                <Tag>v{tenant.config.version}</Tag>
              </Descriptions.Item>
              <Descriptions.Item label="Rule Set ID">
                <Typography.Text code>
                  {tenant.config.ruleSetId}
                </Typography.Text>
              </Descriptions.Item>
              <Descriptions.Item label="Workflow Template">
                <Typography.Text code>
                  {tenant.config.workflowTemplateId}
                </Typography.Text>
              </Descriptions.Item>
            </Descriptions>
          </Card>
        </Col>

        <Col xs={24} lg={12}>
          <Card title="Feature Flags" size="small">
            <Table
              rowKey="key"
              columns={flagColumns}
              dataSource={tenant.config.featureFlags}
              size="small"
              pagination={false}
            />
          </Card>
        </Col>
      </Row>
    </div>
  );
}
