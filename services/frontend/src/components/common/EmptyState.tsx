import { Empty, Typography } from "antd";
import type { ReactNode } from "react";

interface Props {
  icon?: ReactNode;
  title?: string;
  description?: string;
}

export function EmptyState({ title = "Nothing here yet", description }: Props) {
  return (
    <div style={{ padding: "64px 0", textAlign: "center" }}>
      <Empty description={false} />
      <Typography.Title level={5} style={{ marginTop: 16, marginBottom: 4 }}>
        {title}
      </Typography.Title>
      {description && (
        <Typography.Text type="secondary">{description}</Typography.Text>
      )}
    </div>
  );
}
