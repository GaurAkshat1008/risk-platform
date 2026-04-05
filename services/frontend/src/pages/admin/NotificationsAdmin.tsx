import {
  useSendNotificationMutation,
  useUpdateNotificationPreferencesMutation,
  type NotificationChannel
} from "@/graphql/generated";
import { useTenantId } from "@/hooks/useTenantId";
import { showGqlError } from "@/utils/apolloErrors";
import { SendOutlined } from "@ant-design/icons";
import {
  Button,
  Card,
  Form,
  Input,
  Modal,
  Select,
  Switch,
  Table,
  Tag,
  Typography,
  message
} from "antd";
import { useState } from "react";

const CHANNELS: NotificationChannel[] = ["EMAIL", "WEBHOOK", "SLACK"];

export default function NotificationsAdmin() {
  const tenantId = useTenantId();
  const [sendOpen, setSendOpen] = useState(false);
  const [form] = Form.useForm();

  const [send, { loading: sending }] = useSendNotificationMutation({
    onCompleted: (data) => {
      message.success(`Notification sent: ${data.sendNotification.status}`);
      setSendOpen(false);
      form.resetFields();
    },
    onError: showGqlError,
  });

  const [updatePrefs] = useUpdateNotificationPreferencesMutation({
    onCompleted: () => message.success("Preferences updated"),
    onError: showGqlError,
  });

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
          Notifications
        </Typography.Title>
        <Button
          type="primary"
          icon={<SendOutlined />}
          onClick={() => setSendOpen(true)}
        >
          Send Notification
        </Button>
      </div>

      <Card
        title="Channel Preferences"
        size="small"
        style={{ marginBottom: 16 }}
      >
        <Table
          rowKey="channel"
          size="small"
          pagination={false}
          dataSource={CHANNELS.map((ch) => ({ channel: ch }))}
          columns={[
            {
              title: "Channel",
              dataIndex: "channel",
              render: (v: string) => <Tag>{v}</Tag>,
            },
            {
              title: "DECISION_MADE",
              key: "decision",
              width: 120,
              align: "center",
              render: (_, record) => (
                <Switch
                  size="small"
                  defaultChecked
                  onChange={(enabled) =>
                    updatePrefs({
                      variables: {
                        input: {
                          tenantId,
                          channel: record.channel as NotificationChannel,
                          eventType: "DECISION_MADE",
                          enabled,
                          config: "{}",
                        },
                      },
                    })
                  }
                />
              ),
            },
            {
              title: "CASE_ESCALATED",
              key: "escalation",
              width: 130,
              align: "center",
              render: (_, record) => (
                <Switch
                  size="small"
                  defaultChecked
                  onChange={(enabled) =>
                    updatePrefs({
                      variables: {
                        input: {
                          tenantId,
                          channel: record.channel as NotificationChannel,
                          eventType: "CASE_ESCALATED",
                          enabled,
                          config: "{}",
                        },
                      },
                    })
                  }
                />
              ),
            },
            {
              title: "SLA_BREACH",
              key: "sla",
              width: 110,
              align: "center",
              render: (_, record) => (
                <Switch
                  size="small"
                  defaultChecked
                  onChange={(enabled) =>
                    updatePrefs({
                      variables: {
                        input: {
                          tenantId,
                          channel: record.channel as NotificationChannel,
                          eventType: "SLA_BREACH",
                          enabled,
                          config: "{}",
                        },
                      },
                    })
                  }
                />
              ),
            },
          ]}
        />
      </Card>

      <Modal
        title="Send Notification"
        open={sendOpen}
        onCancel={() => setSendOpen(false)}
        confirmLoading={sending}
        onOk={() => form.submit()}
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={(values) =>
            send({
              variables: {
                input: {
                  tenantId,
                  type: values.type,
                  recipient: values.recipient,
                  channel: values.channel,
                  payload: values.payload || "{}",
                },
              },
            })
          }
        >
          <Form.Item
            name="type"
            label="Event Type"
            rules={[{ required: true }]}
          >
            <Select
              options={[
                { value: "DECISION_MADE", label: "Decision Made" },
                { value: "CASE_ESCALATED", label: "Case Escalated" },
                { value: "SLA_BREACH", label: "SLA Breach" },
                { value: "RULE_UPDATED", label: "Rule Updated" },
              ]}
            />
          </Form.Item>
          <Form.Item
            name="recipient"
            label="Recipient"
            rules={[{ required: true }]}
          >
            <Input placeholder="email@example.com or webhook URL" />
          </Form.Item>
          <Form.Item
            name="channel"
            label="Channel"
            rules={[{ required: true }]}
            initialValue="EMAIL"
          >
            <Select options={CHANNELS.map((c) => ({ value: c, label: c }))} />
          </Form.Item>
          <Form.Item name="payload" label="Payload (JSON)">
            <Input.TextArea rows={3} placeholder='{"key": "value"}' />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
