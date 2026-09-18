// Number, time and XP formatting helpers.

const nf = new Intl.NumberFormat('en-US');
const nf1 = new Intl.NumberFormat('en-US', { maximumFractionDigits: 1 });

export function num(n: number | null | undefined): string {
  if (n === null || n === undefined || !Number.isFinite(n)) return '—';
  return nf.format(Math.round(n));
}

export function decimal(n: number | null | undefined, digits = 1): string {
  if (n === null || n === undefined || !Number.isFinite(n)) return '—';
  const opts = { maximumFractionDigits: digits, minimumFractionDigits: 0 };
  return digits === 1
    ? nf1.format(n)
    : new Intl.NumberFormat('en-US', opts).format(n);
}

/** Compact XP: 5.8B, 104.3M, 1.2M, 45.6K, 123. */
export function compact(n: number | null | undefined): string {
  if (n === null || n === undefined || !Number.isFinite(n)) return '—';
  const abs = Math.abs(n);
  if (abs >= 1_000_000_000) return `${nf1.format(n / 1_000_000_000)}B`;
  if (abs >= 1_000_000) return `${nf1.format(n / 1_000_000)}M`;
  if (abs >= 10_000) return `${nf1.format(n / 1_000)}K`;
  return nf.format(Math.round(n));
}

/** Signed gain: +1,234 / -56 / 0 */
export function signed(n: number | null | undefined): string {
  if (n === null || n === undefined || !Number.isFinite(n)) return '—';
  if (n > 0) return `+${nf.format(Math.round(n))}`;
  return nf.format(Math.round(n));
}

/** Compact signed: +1.2M */
export function signedCompact(n: number | null | undefined): string {
  if (n === null || n === undefined || !Number.isFinite(n)) return '—';
  if (n > 0) return `+${compact(n)}`;
  if (n < 0) return `-${compact(Math.abs(n))}`;
  return '0';
}

/** 1,234,567 -> "1,234,567 xp" style already handled by num(); this gives progress text. */
export function pct(n: number | null | undefined, digits = 1): string {
  if (n === null || n === undefined || !Number.isFinite(n)) return '—';
  return `${decimal(n, digits)}%`;
}

/** Hours -> human duration like "3d 4h", "5h 12m", "38m", "just now". */
export function duration(hours: number | null | undefined): string {
  if (hours === null || hours === undefined || !Number.isFinite(hours) || hours <= 0) return '—';
  const totalMin = Math.round(hours * 60);
  if (totalMin < 1) return 'less than a minute';
  const d = Math.floor(totalMin / 1440);
  const h = Math.floor((totalMin % 1440) / 60);
  const m = totalMin % 60;
  if (d > 0) return `${d}d ${h}h`;
  if (h > 0) return `${h}h ${m}m`;
  return `${m}m`;
}

export function timeAgo(iso: string | null | undefined): string {
  if (!iso) return 'never';
  const then = new Date(iso).getTime();
  if (Number.isNaN(then)) return 'never';
  const secs = Math.max(0, Math.round((Date.now() - then) / 1000));
  if (secs < 45) return 'just now';
  const mins = Math.round(secs / 60);
  if (mins < 60) return `${mins}m ago`;
  const hrs = Math.round(mins / 60);
  if (hrs < 24) return `${hrs}h ago`;
  const days = Math.round(hrs / 24);
  if (days < 30) return `${days}d ago`;
  const months = Math.round(days / 30);
  if (months < 12) return `${months}mo ago`;
  return `${Math.round(months / 12)}y ago`;
}

export function dateTime(iso: string | null | undefined): string {
  if (!iso) return '—';
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return '—';
  return d.toLocaleString(undefined, {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });
}

export function dateOnly(iso: string | null | undefined): string {
  if (!iso) return '—';
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return '—';
  return d.toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: 'numeric' });
}
