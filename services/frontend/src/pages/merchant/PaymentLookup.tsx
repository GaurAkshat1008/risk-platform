import { CopyableId } from "@/components/common/CopyableId";
import { EmptyState } from "@/components/common/EmptyState";
import { StatusTag } from "@/components/common/StatusTag";
import { usePaymentEventLazyQuery } from "@/graphql/generated";
import { useTenantId } from "@/hooks/useTenantId";
import { showGqlError } from "@/utils/apolloErrors";
import { formatCurrency, formatTimestamp } from "@/utils/formatters";
import { SearchOutlined } from "@ant-design/icons";
import {
  Button,
  Card,
  Descriptions,
  Input,
  Space,
  Typography
} from "antd";
import { useState } from "react";

export default function PaymentLookup() {
  const tenantId = useTenantId();
  const [key, setKey] = useState("");

  const [lookup, { data, loading, called }] = usePaymentEventLazyQuery({
    onError: showGqlError,
  });

  const handleSearch = () => {
    if (!key.trim()) return;
    lookup({ variables: { tenantId, idempotencyKey: key.trim() } });
  };

  const event = data?.paymentEvent;

  return (
    <div>
      <Typography.Title level={4} style={{ marginBottom: 16 }}>
        Payment Lookup
      </Typography.Title>

      <Card style={{ maxWidth: 700 }}>
        <Space.Compact style={{ width: "100%", marginBottom: 24 }}>
          <Input
            placeholder="Enter idempotency key"
            value={key}
            onChange={(e) => setKey(e.target.value)}
            onPressEnter={handleSearch}
            allowClear
          />
          <Button
            type="primary"
            icon={<SearchOutlined />}
            onClick={handleSearch}
            loading={loading}
          >
            Search
          </Button>
        </Space.Compact>

        {called && !loading && !event && (
          <EmptyState
            title="No payment found"
            description="Check the idempotency key and try again."
          />
        )}

        {event && (
          <Descriptions
            bordered
            size="small"
            column={1}
            labelStyle={{ width: 160, fontWeight: 500, fontSize: 12 }}
          >
            <Descriptions.Item label="Event ID">
              <CopyableId id={event.id} />
            </Descriptions.Item>
            <Descriptions.Item label="Idempotency Key">
              <Typography.Text code>{event.idempotencyKey}</Typography.Text>
            </Descriptions.Item>
            <Descriptions.Item label="Status">
              <StatusTag status={event.status} />
            </Descriptions.Item>
            <Descriptions.Item label="Amount">
              {formatCurrency(event.amount)} {event.currency}
            </Descriptions.Item>
            <Descriptions.Item label="Source">
              <Typography.Text code>{event.source}</Typography.Text>
            </Descriptions.Item>
            <Descriptions.Item label="Destination">
              <Typography.Text code>{event.destination}</Typography.Text>
            </Descriptions.Item>
            <Descriptions.Item label="Received At">
              {formatTimestamp(event.receivedAt)}
            </Descriptions.Item>
          </Descriptions>
        )}
      </Card>
    </div>
  );
}
