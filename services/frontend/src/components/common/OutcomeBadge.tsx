import { Tag } from "antd";

const outcomeStyles: Record<string, { color: string; label: string }> = {
  APPROVE: { color: "#16a34a", label: "Approve" },
  FLAG: { color: "#d97706", label: "Flag" },
  REVIEW: { color: "#2563eb", label: "Review" },
  BLOCK: { color: "#dc2626", label: "Block" },
};

interface Props {
  outcome: string;
}

export function OutcomeBadge({ outcome }: Props) {
  const style = outcomeStyles[outcome] ?? { color: "#6b7280", label: outcome };
  return (
    <Tag
      color={style.color}
      style={{
        fontWeight: 600,
        fontSize: 11,
        letterSpacing: "0.04em",
        textTransform: "uppercase",
      }}
    >
      {style.label}
    </Tag>
  );
}
