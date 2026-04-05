import { Typography, Tooltip, message } from "antd";
import { CopyOutlined } from "@ant-design/icons";
import { truncateId } from "@/utils/formatters";

interface Props {
  id: string;
  maxLen?: number;
}

export function CopyableId({ id, maxLen = 8 }: Props) {
  const copy = () => {
    navigator.clipboard.writeText(id);
    message.success("Copied");
  };

  return (
    <Tooltip title={id}>
      <Typography.Text
        code
        style={{ fontSize: 12, cursor: "pointer" }}
        onClick={copy}
      >
        {truncateId(id, maxLen)}{" "}
        <CopyOutlined style={{ fontSize: 10, opacity: 0.5 }} />
      </Typography.Text>
    </Tooltip>
  );
}
