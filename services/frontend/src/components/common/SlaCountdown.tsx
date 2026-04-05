import { useEffect, useState } from "react";
import { Tag, Tooltip } from "antd";
import dayjs from "dayjs";

interface Props {
  deadline: string | Date;
}

/**
 * Real-time SLA countdown badge.
 * - Green  → > 50% time remaining
 * - Yellow → 10-50% remaining
 * - Red    → < 10% or breached
 */
export function SlaCountdown({ deadline }: Props) {
  const [now, setNow] = useState(dayjs());

  useEffect(() => {
    const id = setInterval(() => setNow(dayjs()), 1000);
    return () => clearInterval(id);
  }, []);

  const end = dayjs(deadline);
  const totalSec = end.diff(dayjs(deadline).subtract(1, "day"), "second"); // approximate SLA window
  const remainSec = end.diff(now, "second");

  if (remainSec <= 0) {
    return (
      <Tooltip title={`Breached at ${end.format("HH:mm:ss")}`}>
        <Tag
          color="red"
          style={{ fontVariantNumeric: "tabular-nums", fontWeight: 600 }}
        >
          BREACHED
        </Tag>
      </Tooltip>
    );
  }

  const pct = totalSec > 0 ? remainSec / totalSec : 1;
  const color = pct > 0.5 ? "green" : pct > 0.1 ? "orange" : "red";

  const hours = Math.floor(remainSec / 3600);
  const mins = Math.floor((remainSec % 3600) / 60);
  const secs = remainSec % 60;
  const label = hours > 0 ? `${hours}h ${mins}m` : `${mins}m ${secs}s`;

  return (
    <Tooltip title={`SLA deadline: ${end.format("YYYY-MM-DD HH:mm:ss")}`}>
      <Tag
        color={color}
        style={{
          fontVariantNumeric: "tabular-nums",
          fontWeight: 600,
          minWidth: 64,
          textAlign: "center",
        }}
      >
        {label}
      </Tag>
    </Tooltip>
  );
}
