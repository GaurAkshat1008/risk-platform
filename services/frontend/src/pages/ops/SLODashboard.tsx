import { CardSkeleton } from "@/components/common/PageSkeleton";
import { useSloStatusQuery, type SLOStatus } from "@/graphql/generated";
import { usePolling } from "@/hooks/usePolling";
import {
  ArrowDownOutlined,
  ArrowUpOutlined,
  CheckCircleOutlined,
  ClockCircleOutlined,
} from "@ant-design/icons";
import { Card, Col, Row, Select, Space, Statistic, Typography } from "antd";
import { useState } from "react";
import {
  Bar,
  BarChart,
  CartesianGrid,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";

const SERVICES = [
  "ingestion",
  "decision",
  "rules-engine",
  "case-management",
  "workflow",
  "identity-access",
  "tenant-config",
  "notification",
  "audit-trail",
  "explanation",
  "graphql-bff",
];

const WINDOWS = ["1h", "6h", "24h", "7d", "30d"];

function AvailabilityCard({ slo }: { slo: SLOStatus }) {
  const pct = slo.availability * 100;
  const isGood = pct >= 99.9;
  return (
    <Card size="small">
      <Statistic
        title="Availability"
        value={pct}
        precision={3}
        suffix="%"
        valueStyle={{ color: isGood ? "#16a34a" : "#dc2626" }}
        prefix={isGood ? <CheckCircleOutlined /> : <ArrowDownOutlined />}
      />
    </Card>
  );
}

function ErrorRateCard({ slo }: { slo: SLOStatus }) {
  const isOk = slo.errorRate < 0.01;
  return (
    <Card size="small">
      <Statistic
        title="Error Rate"
        value={slo.errorRate * 100}
        precision={3}
        suffix="%"
        valueStyle={{ color: isOk ? "#16a34a" : "#dc2626" }}
        prefix={isOk ? <ArrowDownOutlined /> : <ArrowUpOutlined />}
      />
    </Card>
  );
}

function LatencyCard({ slo }: { slo: SLOStatus }) {
  return (
    <Card size="small">
      <Statistic
        title="P99 Latency"
        value={slo.p99LatencyMs}
        precision={1}
        suffix="ms"
        prefix={<ClockCircleOutlined />}
        valueStyle={{
          color:
            slo.p99LatencyMs > 500
              ? "#dc2626"
              : slo.p99LatencyMs > 200
                ? "#d97706"
                : "#16a34a",
        }}
      />
    </Card>
  );
}

export default function SLODashboard() {
  const [service, setService] = useState("decision");
  const [window, setWindow] = useState("1h");

  const { data, loading, refetch } = useSloStatusQuery({
    variables: { service, window },
  });

  usePolling(() => refetch(), 30000);

  if (loading && !data) return <CardSkeleton />;

  const slo = data?.sloStatus;
  const latencyData = slo
    ? [
        { name: "P50", value: slo.p50LatencyMs },
        { name: "P95", value: slo.p95LatencyMs },
        { name: "P99", value: slo.p99LatencyMs },
      ]
    : [];

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
          SLO Dashboard
        </Typography.Title>
        <Space>
          <Select
            value={service}
            onChange={setService}
            style={{ width: 180 }}
            options={SERVICES.map((s) => ({ value: s, label: s }))}
          />
          <Select
            value={window}
            onChange={setWindow}
            style={{ width: 100 }}
            options={WINDOWS.map((w) => ({ value: w, label: w }))}
          />
        </Space>
      </div>

      {slo && (
        <>
          <Row gutter={[16, 16]} style={{ marginBottom: 16 }}>
            <Col xs={24} sm={8}>
              <AvailabilityCard slo={slo} />
            </Col>
            <Col xs={24} sm={8}>
              <ErrorRateCard slo={slo} />
            </Col>
            <Col xs={24} sm={8}>
              <LatencyCard slo={slo} />
            </Col>
          </Row>

          <Card title="Latency Distribution" size="small">
            <ResponsiveContainer width="100%" height={260}>
              <BarChart
                data={latencyData}
                margin={{ top: 8, right: 16, bottom: 0, left: 0 }}
              >
                <CartesianGrid strokeDasharray="3 3" stroke="#e9ecef" />
                <XAxis dataKey="name" tick={{ fontSize: 12 }} />
                <YAxis tick={{ fontSize: 12 }} unit="ms" />
                <Tooltip
                  formatter={(value: number) => [
                    `${value.toFixed(1)}ms`,
                    "Latency",
                  ]}
                  contentStyle={{ fontSize: 12, borderRadius: 4 }}
                />
                <Bar dataKey="value" fill="#475569" radius={[4, 4, 0, 0]} />
              </BarChart>
            </ResponsiveContainer>
          </Card>
        </>
      )}
    </div>
  );
}
