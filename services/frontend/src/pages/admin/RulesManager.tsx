import { OutcomeBadge } from "@/components/common/OutcomeBadge";
import { TableSkeleton } from "@/components/common/PageSkeleton";
import { RelativeTime } from "@/components/common/RelativeTime";
import {
  useCreateRuleMutation,
  useDeleteRuleMutation,
  useRulesQuery,
  useSimulateRuleMutation,
  useUpdateRuleMutation,
  type Rule,
} from "@/graphql/generated";
import { useTenantId } from "@/hooks/useTenantId";
import { showGqlError } from "@/utils/apolloErrors";
import {
  DeleteOutlined,
  PlayCircleOutlined,
  PlusOutlined,
} from "@ant-design/icons";
import {
  Button,
  Card,
  Form,
  Input,
  InputNumber,
  Modal,
  Select,
  Space,
  Switch,
  Table,
  Tag,
  Typography,
  message,
} from "antd";
import type { ColumnsType } from "antd/es/table";
import { useState } from "react";

export default function RulesManager() {
  const tenantId = useTenantId();
  const [showDisabled, setShowDisabled] = useState(true);
  const [createOpen, setCreateOpen] = useState(false);
  const [simOpen, setSimOpen] = useState(false);
  const [simRuleId, setSimRuleId] = useState("");

  const { data, loading, refetch } = useRulesQuery({
    variables: { tenantId, includeDisabled: showDisabled },
  });

  const [createRule, { loading: creating }] = useCreateRuleMutation({
    onCompleted: () => {
      message.success("Rule created");
      refetch();
      setCreateOpen(false);
    },
    onError: showGqlError,
  });

  const [updateRule] = useUpdateRuleMutation({
    onCompleted: () => {
      message.success("Rule updated");
      refetch();
    },
    onError: showGqlError,
  });

  const [deleteRule] = useDeleteRuleMutation({
    onCompleted: () => {
      message.success("Rule deleted");
      refetch();
    },
    onError: showGqlError,
  });

  const [simulateRule, { data: simData, loading: simulating }] =
    useSimulateRuleMutation({
      onError: showGqlError,
    });

  const [createForm] = Form.useForm();
  const [simForm] = Form.useForm();

  const columns: ColumnsType<Rule> = [
    {
      title: "Name",
      dataIndex: "name",
      render: (v: string, r: Rule) => (
        <Space>
          <Typography.Text strong>{v}</Typography.Text>
          <Tag style={{ fontSize: 10 }}>v{r.version}</Tag>
        </Space>
      ),
    },
    {
      title: "Action",
      dataIndex: "action",
      width: 90,
      render: (v: string) => <OutcomeBadge outcome={v} />,
    },
    {
      title: "Priority",
      dataIndex: "priority",
      width: 80,
      align: "right",
      sorter: (a, b) => a.priority - b.priority,
    },
    {
      title: "Enabled",
      dataIndex: "enabled",
      width: 80,
      align: "center",
      render: (v: boolean, record: Rule) => (
        <Switch
          size="small"
          checked={v}
          onChange={(checked) =>
            updateRule({
              variables: {
                input: {
                  ruleId: record.id,
                  tenantId,
                  expression: "",
                  action: record.action,
                  priority: record.priority,
                  enabled: checked,
                },
              },
            })
          }
        />
      ),
    },
    {
      title: "Updated",
      dataIndex: "updatedAt",
      width: 130,
      render: (v: string) => <RelativeTime timestamp={v} />,
    },
    {
      title: "",
      key: "actions",
      width: 100,
      render: (_, record) => (
        <Space size="small">
          <Button
            type="text"
            size="small"
            icon={<PlayCircleOutlined />}
            onClick={() => {
              setSimRuleId(record.id);
              setSimOpen(true);
            }}
          />
          <Button
            type="text"
            size="small"
            danger
            icon={<DeleteOutlined />}
            onClick={() =>
              Modal.confirm({
                title: `Delete rule "${record.name}"?`,
                onOk: () =>
                  deleteRule({
                    variables: { input: { ruleId: record.id, tenantId } },
                  }),
              })
            }
          />
        </Space>
      ),
    },
  ];

  if (loading && !data) return <TableSkeleton rows={8} />;

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
          Rules Manager
        </Typography.Title>
        <Space>
          <Switch
            checkedChildren="Show disabled"
            unCheckedChildren="Active only"
            checked={showDisabled}
            onChange={setShowDisabled}
          />
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => setCreateOpen(true)}
          >
            Create Rule
          </Button>
        </Space>
      </div>

      <Card bodyStyle={{ padding: 0 }}>
        <Table
          rowKey="id"
          columns={columns}
          dataSource={data?.rules ?? []}
          loading={loading}
          size="small"
          pagination={{ pageSize: 15, showSizeChanger: false }}
        />
      </Card>

      {/* Create Rule Modal */}
      <Modal
        title="Create Rule"
        open={createOpen}
        onCancel={() => setCreateOpen(false)}
        confirmLoading={creating}
        onOk={() => createForm.submit()}
      >
        <Form
          form={createForm}
          layout="vertical"
          onFinish={(values) =>
            createRule({ variables: { input: { ...values, tenantId } } })
          }
        >
          <Form.Item name="name" label="Name" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item
            name="expression"
            label="Expression"
            rules={[{ required: true }]}
          >
            <Input.TextArea
              rows={3}
              placeholder='e.g. amount > 100000 && country == "NG"'
            />
          </Form.Item>
          <Space>
            <Form.Item
              name="action"
              label="Action"
              rules={[{ required: true }]}
              initialValue="FLAG"
            >
              <Select
                style={{ width: 120 }}
                options={["APPROVE", "FLAG", "REVIEW", "BLOCK"].map((a) => ({
                  value: a,
                  label: a,
                }))}
              />
            </Form.Item>
            <Form.Item
              name="priority"
              label="Priority"
              rules={[{ required: true }]}
              initialValue={100}
            >
              <InputNumber min={1} max={9999} />
            </Form.Item>
          </Space>
        </Form>
      </Modal>

      {/* Simulate Rule Modal */}
      <Modal
        title="Simulate Rule"
        open={simOpen}
        onCancel={() => {
          setSimOpen(false);
          setSimRuleId("");
        }}
        confirmLoading={simulating}
        onOk={() => simForm.submit()}
        width={500}
      >
        <Form
          form={simForm}
          layout="vertical"
          onFinish={(values) =>
            simulateRule({
              variables: {
                input: {
                  tenantId,
                  ruleId: simRuleId,
                  expression: "",
                  action: "FLAG",
                  ...values,
                },
              },
            })
          }
        >
          <Form.Item
            name="paymentEventId"
            label="Payment Event ID"
            rules={[{ required: true }]}
          >
            <Input />
          </Form.Item>
          <Space>
            <Form.Item
              name="amount"
              label="Amount (cents)"
              rules={[{ required: true }]}
            >
              <InputNumber min={1} />
            </Form.Item>
            <Form.Item name="currency" label="Currency" initialValue="USD">
              <Input style={{ width: 80 }} />
            </Form.Item>
          </Space>
          <Form.Item name="source" label="Source" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item
            name="destination"
            label="Destination"
            rules={[{ required: true }]}
          >
            <Input />
          </Form.Item>
        </Form>

        {simData?.simulateRule && (
          <Card
            size="small"
            style={{
              marginTop: 16,
              background: "var(--ant-color-fill-quaternary)",
            }}
          >
            <Space direction="vertical" size={4}>
              <Typography.Text>
                <strong>Matched:</strong>{" "}
                {simData.simulateRule.matched ? "Yes" : "No"}
              </Typography.Text>
              <Typography.Text>
                <strong>Action:</strong>{" "}
                <OutcomeBadge outcome={simData.simulateRule.action} />
              </Typography.Text>
              <Typography.Text>
                <strong>Reason:</strong> {simData.simulateRule.reason}
              </Typography.Text>
            </Space>
          </Card>
        )}
      </Modal>
    </div>
  );
}
