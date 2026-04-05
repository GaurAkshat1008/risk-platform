import { Tag } from "antd";

const statusMap: Record<string, { color: string; label: string }> = {
  OPEN: { color: "blue", label: "Open" },
  IN_REVIEW: { color: "orange", label: "In Review" },
  RESOLVED: { color: "green", label: "Resolved" },
  ESCALATED: { color: "red", label: "Escalated" },
  PENDING: { color: "default", label: "Pending" },
  DELIVERED: { color: "green", label: "Delivered" },
  FAILED: { color: "red", label: "Failed" },
  RETRYING: { color: "orange", label: "Retrying" },
  ACTIVE: { color: "green", label: "Active" },
  INACTIVE: { color: "default", label: "Inactive" },
  ONBOARDING: { color: "blue", label: "Onboarding" },
};

interface Props {
  status: string;
}

export function StatusTag({ status }: Props) {
  const entry = statusMap[status] ?? { color: "default", label: status };
  return <Tag color={entry.color}>{entry.label}</Tag>;
}
