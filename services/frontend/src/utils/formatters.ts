import dayjs from 'dayjs';

/** Format cents as currency string, e.g. 150000 → "$1,500.00" */
export function formatCurrency(amountCents: number, currency = 'USD'): string {
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency,
    minimumFractionDigits: 2,
  }).format(amountCents / 100);
}

/** Format percentage, e.g. 0.9954 → "99.54%" */
export function formatPercent(value: number, decimals = 2): string {
  return `${(value * 100).toFixed(decimals)}%`;
}

/** Format a timestamp as "2025-01-15 14:30:05" */
export function formatTimestamp(ts: string | Date): string {
  return dayjs(ts).format('YYYY-MM-DD HH:mm:ss');
}

/** Format duration in ms as "123ms" or "1.2s" */
export function formatDuration(ms: number): string {
  if (ms < 1000) return `${ms}ms`;
  return `${(ms / 1000).toFixed(1)}s`;
}

/** Truncate a UUID to first 8 chars: "a1b2c3d4-..." → "a1b2c3d4" */
export function truncateId(id: string, len = 8): string {
  return id.length > len ? id.slice(0, len) : id;
}
