import { Skeleton, Space } from "antd";

/** Full-page loading skeleton that mirrors table + header layout. */
export function PageSkeleton() {
  return (
    <Space
      direction="vertical"
      size="large"
      style={{ width: "100%", padding: "24px 0" }}
    >
      <Skeleton.Input active size="small" style={{ width: 200 }} />
      <Skeleton active paragraph={{ rows: 1, width: ["60%"] }} title={false} />
      <Skeleton active paragraph={{ rows: 8 }} title={false} />
    </Space>
  );
}

/** Compact inline skeleton for cards / description panels. */
export function CardSkeleton() {
  return <Skeleton active paragraph={{ rows: 4 }} />;
}

/** Table-shaped skeleton. */
export function TableSkeleton({ rows = 6 }: { rows?: number }) {
  return <Skeleton active paragraph={{ rows }} title={false} />;
}
