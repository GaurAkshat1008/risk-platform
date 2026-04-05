import { CopyableId } from "@/components/common/CopyableId";
import { CardSkeleton } from "@/components/common/PageSkeleton";
import { RelativeTime } from "@/components/common/RelativeTime";
import {
  useCreateWorkflowTemplateMutation,
  useWorkflowTemplatesQuery,
  type WorkflowTemplate,
  type WorkflowTransition,
} from "@/graphql/generated";
import { useTenantId } from "@/hooks/useTenantId";
import { showGqlError } from "@/utils/apolloErrors";
import { PlusOutlined } from "@ant-design/icons";
import {
  Button,
  Card,
  Col,
  Descriptions,
  Form,
  Input,
  Modal,
  Row,
  Space,
  Table,
  Tag,
  Typography,
  message
} from "antd";
import type { ColumnsType } from "antd/es/table";
import { useState } from "react";

export default function WorkflowBuilder() {
  const tenantId = useTenantId();
  const [createOpen, setCreateOpen] = useState(false);
  const [selected, setSelected] = useState<WorkflowTemplate | null>(null);

  const { data, loading, refetch } = useWorkflowTemplatesQuery({
    variables: { tenantId },
  });

  const [create, { loading: creating }] = useCreateWorkflowTemplateMutation({
    onCompleted: () => {
      message.success("Workflow created");
      refetch();
      setCreateOpen(false);
    },
    onError: showGqlError,
  });

  const [form] = Form.useForm();

  const templateColumns: ColumnsType<WorkflowTemplate> = [
    {
      title: "Name",
      dataIndex: "name",
      render: (v: string, r: WorkflowTemplate) => (
        <Typography.Link onClick={() => setSelected(r)}>
          {v} <Tag style={{ fontSize: 10 }}>v{r.version}</Tag>
        </Typography.Link>
      ),
    },
    {
      title: "States",
      dataIndex: "states",
      render: (states: string[]) => (
        <Space size={4} wrap>
          {states.map((s) => (
            <Tag key={s}>{s}</Tag>
          ))}
        </Space>
      ),
    },
    {
      title: "Transitions",
      dataIndex: "transitions",
      width: 100,
      align: "center",
      render: (t: WorkflowTransition[]) => t.length,
    },
    {
      title: "Updated",
      dataIndex: "updatedAt",
      width: 130,
      render: (v: string) => <RelativeTime timestamp={v} />,
    },
  ];

  const transitionColumns: ColumnsType<WorkflowTransition> = [
    { title: "From", dataIndex: "fromState", width: 120 },
    { title: "To", dataIndex: "toState", width: 120 },
    { title: "Required Role", dataIndex: "requiredRole", width: 150 },
    {
      title: "Guards",
      dataIndex: "guards",
      render: (guards: { type: string; role: string; condition: string }[]) =>
        guards.map((g, i) => (
          <Tag key={i} style={{ fontSize: 11 }}>
            {g.type}: {g.condition || g.role}
          </Tag>
        )),
    },
  ];

  if (loading && !data) return <CardSkeleton />;

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
          Workflow Builder
        </Typography.Title>
        <Button
          type="primary"
          icon={<PlusOutlined />}
          onClick={() => setCreateOpen(true)}
        >
          New Workflow
        </Button>
      </div>

      <Row gutter={[16, 16]}>
        <Col xs={24} lg={selected ? 10 : 24}>
          <Card bodyStyle={{ padding: 0 }}>
            <Table
              rowKey="id"
              columns={templateColumns}
              dataSource={data?.workflowTemplates ?? []}
              size="small"
              pagination={false}
              onRow={(record) => ({
                style: {
                  cursor: "pointer",
                  background:
                    selected?.id === record.id
                      ? "var(--ant-color-fill-quaternary)"
                      : undefined,
                },
                onClick: () => setSelected(record),
              })}
            />
          </Card>
        </Col>

        {selected && (
          <Col xs={24} lg={14}>
            <Card
              title={`${selected.name} v${selected.version}`}
              size="small"
              extra={
                <Button
                  type="text"
                  size="small"
                  onClick={() => setSelected(null)}
                >
                  Close
                </Button>
              }
            >
              <Descriptions
                column={1}
                size="small"
                style={{ marginBottom: 16 }}
                labelStyle={{ fontSize: 12, fontWeight: 500, width: 110 }}
              >
                <Descriptions.Item label="ID">
                  <CopyableId id={selected.id} />
                </Descriptions.Item>
                <Descriptions.Item label="States">
                  <Space size={4} wrap>
                    {selected.states.map((s) => (
                      <Tag key={s}>{s}</Tag>
                    ))}
                  </Space>
                </Descriptions.Item>
                <Descriptions.Item label="Created">
                  <RelativeTime timestamp={selected.createdAt} />
                </Descriptions.Item>
              </Descriptions>

              <Typography.Text
                strong
                style={{ fontSize: 12, display: "block", marginBottom: 8 }}
              >
                Transitions
              </Typography.Text>
              <Table
                rowKey={(r) => `${r.fromState}-${r.toState}`}
                columns={transitionColumns}
                dataSource={selected.transitions}
                size="small"
                pagination={false}
              />
            </Card>
          </Col>
        )}
      </Row>

      {/* Create Modal */}
      <Modal
        title="Create Workflow Template"
        open={createOpen}
        onCancel={() => setCreateOpen(false)}
        confirmLoading={creating}
        onOk={() => form.submit()}
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={(values) =>
            create({
              variables: {
                input: {
                  tenantId,
                  name: values.name,
                  states: values.states.split(",").map((s: string) => s.trim()),
                  transitions: [],
                },
              },
            })
          }
        >
          <Form.Item name="name" label="Name" rules={[{ required: true }]}>
            <Input placeholder="e.g. Default Case Workflow" />
          </Form.Item>
          <Form.Item
            name="states"
            label="States (comma-separated)"
            rules={[{ required: true }]}
          >
            <Input placeholder="OPEN, IN_REVIEW, RESOLVED, ESCALATED" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
