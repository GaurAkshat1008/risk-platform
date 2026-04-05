import { OutcomeBadge } from "@/components/common/OutcomeBadge";
import { useIngestPaymentMutation } from "@/graphql/generated";
import { useTenantId } from "@/hooks/useTenantId";
import { showGqlError } from "@/utils/apolloErrors";
import { ReloadOutlined, SendOutlined } from "@ant-design/icons";
import {
  Button,
  Card,
  Descriptions,
  Form,
  Input,
  InputNumber,
  Result,
  Select,
  Space,
  Typography,
} from "antd";
import { useState } from "react";
interface FormValues {
  idempotencyKey: string;
  amount: number;
  currency: string;
  source: string;
  destination: string;
}

export default function PaymentSubmit() {
  const tenantId = useTenantId();
  const [form] = Form.useForm<FormValues>();
  const [submitted, setSubmitted] = useState<{
    eventId: string;
    status: string;
    reason: string;
  } | null>(null);

  const [ingest, { loading }] = useIngestPaymentMutation({
    onCompleted: (data) => setSubmitted(data.ingestPayment),
    onError: showGqlError,
  });

  const handleSubmit = (values: FormValues) => {
    ingest({
      variables: {
        input: {
          idempotencyKey: values.idempotencyKey,
          tenantId,
          amount: Math.round(values.amount * 100),
          currency: values.currency,
          source: values.source,
          destination: values.destination,
        },
      },
    });
  };

  const handleReset = () => {
    form.resetFields();
    setSubmitted(null);
  };

  if (submitted) {
    return (
      <Card style={{ maxWidth: 600 }}>
        <Result
          status={submitted.status === "RECEIVED" ? "success" : "warning"}
          title="Payment Submitted"
          subTitle={submitted.reason}
          extra={[
            <Button
              key="new"
              type="primary"
              icon={<ReloadOutlined />}
              onClick={handleReset}
            >
              Submit Another
            </Button>,
          ]}
        >
          <Descriptions column={1} size="small" bordered>
            <Descriptions.Item label="Event ID">
              <Typography.Text copyable code>
                {submitted.eventId}
              </Typography.Text>
            </Descriptions.Item>
            <Descriptions.Item label="Status">
              <OutcomeBadge outcome={submitted.status as "APPROVE"} />
            </Descriptions.Item>
          </Descriptions>
        </Result>
      </Card>
    );
  }

  return (
    <div>
      <Typography.Title level={4} style={{ marginBottom: 16 }}>
        Submit Payment
      </Typography.Title>

      <Card style={{ maxWidth: 600 }}>
        <Form
          form={form}
          layout="vertical"
          onFinish={handleSubmit}
          initialValues={{ currency: "USD" }}
          requiredMark="optional"
          size="middle"
        >
          <Form.Item
            name="idempotencyKey"
            label="Idempotency Key"
            rules={[{ required: true, message: "Required" }]}
            tooltip="Unique key to prevent duplicate submissions"
          >
            <Input placeholder="e.g. pay_abc123_20250101" />
          </Form.Item>

          <Space size="middle" style={{ width: "100%" }}>
            <Form.Item
              name="amount"
              label="Amount"
              rules={[{ required: true, message: "Required" }]}
              style={{ flex: 1 }}
            >
              <InputNumber
                min={0.01}
                step={0.01}
                precision={2}
                style={{ width: "100%" }}
                addonBefore="$"
              />
            </Form.Item>

            <Form.Item
              name="currency"
              label="Currency"
              rules={[{ required: true }]}
              style={{ width: 120 }}
            >
              <Select
                options={[
                  { value: "USD", label: "USD" },
                  { value: "EUR", label: "EUR" },
                  { value: "GBP", label: "GBP" },
                ]}
              />
            </Form.Item>
          </Space>

          <Form.Item
            name="source"
            label="Source Account"
            rules={[{ required: true, message: "Required" }]}
          >
            <Input placeholder="Source identifier" />
          </Form.Item>

          <Form.Item
            name="destination"
            label="Destination Account"
            rules={[{ required: true, message: "Required" }]}
          >
            <Input placeholder="Destination identifier" />
          </Form.Item>

          <Form.Item style={{ marginBottom: 0 }}>
            <Button
              type="primary"
              htmlType="submit"
              loading={loading}
              icon={<SendOutlined />}
              block
            >
              Ingest Payment
            </Button>
          </Form.Item>
        </Form>
      </Card>
    </div>
  );
}
