import { Tooltip } from "antd";
import dayjs from "dayjs";
import relativeTime from "dayjs/plugin/relativeTime";

dayjs.extend(relativeTime);

interface Props {
  timestamp: string | Date;
}

export function RelativeTime({ timestamp }: Props) {
  const d = dayjs(timestamp);
  return (
    <Tooltip title={d.format("YYYY-MM-DD HH:mm:ss")}>
      <span style={{ whiteSpace: "nowrap" }}>{d.fromNow()}</span>
    </Tooltip>
  );
}
