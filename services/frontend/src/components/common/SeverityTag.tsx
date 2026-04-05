import { Tag } from "antd";

const severityColors: Record<string, string> = {
  DEBUG: "#868e96",
  INFO: "#2563eb",
  WARN: "#d97706",
  ERROR: "#dc2626",
  FATAL: "#7c3aed",
};

interface Props {
  severity: string;
}

export function SeverityTag({ severity }: Props) {
  const upper = severity.toUpperCase();
  return (
    <Tag
      color={severityColors[upper] ?? "#868e96"}
      style={{ fontSize: 11, fontWeight: 600 }}
    >
      {upper}
    </Tag>
  );
}
