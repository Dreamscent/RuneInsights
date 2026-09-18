<script lang="ts">
  import { store } from '../lib/store.svelte';
  import { api, type Rates, type HistoryResponse, type Period, type SkillRatesResponse } from '../lib/api';
  import SkillIcon from '../components/SkillIcon.svelte';
  import ProgressBar from '../components/ProgressBar.svelte';
  import Spinner from '../components/Spinner.svelte';
  import LineChart from '../components/LineChart.svelte';
  import { compact, decimal, duration, num, signedCompact, dateTime } from '../lib/format';
  import { skillColor } from '../lib/skills';

  const PERIODS: Period[] = ['day', 'week', 'month', 'year'];

  let rates = $state<Rates | null>(null);
  let loadingRates = $state(true);
  let period = $state<Period>('week');

  let skill = $state('overall');
  let rangeDays = $state(30);
  let history = $state<HistoryResponse | null>(null);
  let loadingHistory = $state(true);

  let storeRates = $state<SkillRatesResponse | null>(null);

  async function loadRates(id: number, p: Period) {
    loadingRates = true;
    try {
      const [r, sr] = await Promise.all([api.rates(id, p), api.skillRates()]);
      rates = r;
      storeRates = sr;
    } catch (e) {
      store.notify(e instanceof Error ? e.message : 'Failed to load rates', 'err');
    } finally {
      loadingRates = false;
    }
  }

  async function loadHistory(id: number, sk: string, days: number) {
    loadingHistory = true;
    try {
      const to = new Date();
      const from = new Date(to.getTime() - days * 86_400_000);
      history = await api.history(id, sk, from.toISOString(), to.toISOString());
    } catch (e) {
      store.notify(e instanceof Error ? e.message : 'Failed to load history', 'err');
    } finally {
      loadingHistory = false;
    }
  }

  $effect(() => {
    const id = store.selectedId;
    if (id == null) return;
    const p = period;
    const sk = skill;
    const rd = rangeDays;
    void loadRates(id, p);
    void loadHistory(id, sk, rd);
  });

  const sorted = $derived.by(() => {
    const list = [...(rates?.skills ?? [])];
    return list.sort((a, b) => {
      if (a.hasBaseline !== b.hasBaseline) return a.hasBaseline ? -1 : 1;
      return b.gain - a.gain;
    });
  });

  const maxGain = $derived(
    Math.max(1, ...(rates?.skills ?? []).map((s) => s.gain)),
  );

  const chartLabels = $derived(
    (history?.points ?? []).map((p) =>
      new Date(p.at).toLocaleString(undefined, {
        month: 'short',
        day: 'numeric',
        hour: rangeDays <= 2 ? '2-digit' : undefined,
        minute: rangeDays <= 2 ? '2-digit' : undefined,
      }),
    ),
  );

  // daily deltas between consecutive snapshots, for a "gain over time" feel
  const chartPoints = $derived.by(() => {
    const pts = history?.points ?? [];
    const deltas: number[] = [];
    for (let i = 1; i < pts.length; i++) deltas.push(Math.max(0, pts[i].xp - pts[i - 1].xp));
    if (pts.length > 0) {
      // prepend the balance of the first point as baseline
      deltas.unshift(0);
    }
    return deltas;
  });

  const skillOptions = $derived.by(() => {
    const opts: { key: string; name: string }[] = [{ key: 'overall', name: 'Overall (all skills)' }];
    for (const s of store.skills?.skills ?? []) opts.push({ key: s.key, name: s.name });
    const current = store.skills?.skills.find((x) => x.key === skill);
    if (skill !== 'overall' && !current) opts.push({ key: skill, name: skill });
    return opts;
  });

  const paceRows = $derived.by(() => {
    const assumed = new Map<string, { perHour: number; method: string }>();
    for (const row of storeRates?.skills ?? []) {
      assumed.set(row.key, { perHour: row.methods[0]?.perHour ?? 0, method: row.methods[0]?.method ?? '' });
    }
    const out: { key: string; name: string; assumed: number; measured: number; efficiency: number; bg: string; fg: string }[] = [];
    if (!rates?.hasBaseline || rates.elapsedHours <= 0) {
      return out;
    }
    for (const sg of rates.skills ?? []) {
      const a = assumed.get(sg.key)?.perHour ?? 0;
      if (a <= 0 || !sg.hasBaseline || sg.gain <= 0) continue;
      const measured = (sg.gain / rates.elapsedHours) as number;
      const efficiency = Math.max(0, Math.min(999, (measured / a) * 100));
      const good = efficiency >= 100;
      const mid = efficiency >= 70;
      out.push({
        key: sg.key,
        name: sg.name,
        assumed: a,
        measured,
        efficiency,
        bg: good ? 'rgba(52,211,153,.14)' : mid ? 'var(--color-bg-soft)' : 'rgba(251,113,133,.14)',
        fg: good ? 'var(--color-jade)' : mid ? 'var(--color-muted)' : 'var(--color-rose)',
      });
    }
    return out;
  });

  const RANGES = [
    { days: 1, label: '24h' },
    { days: 7, label: '7d' },
    { days: 30, label: '30d' },
    { days: 90, label: '90d' },
    { days: 365, label: '1y' },
  ];
</script>

<div class="flex flex-col gap-5">
  {#if !store.skills}
    <div class="flex justify-center py-16"><Spinner /></div>
  {:else}
    <!-- period tabs -->
    <div class="flex items-center gap-2">
      <div class="card inline-flex overflow-hidden p-1">
        {#each PERIODS as p (p)}
          <button
            class="rounded-lg px-4 py-1.5 text-xs font-medium capitalize transition-colors
              {period === p
              ? 'bg-[var(--color-panel-2)] text-[var(--color-gold-soft)] shadow-[inset_0_0_0_1px_var(--color-line-2)]'
              : 'text-[var(--color-muted)] hover:text-[var(--color-ink)]'}"
            onclick={() => (period = p)}
          >
            {p}
          </button>
        {/each}
      </div>
      <span class="text-xs text-[var(--color-faint)]">
        {#if rates?.hasBaseline}
          {dateTime(rates.from)} → {dateTime(rates.to)} ({duration(rates.elapsedHours)})
        {:else}
          need a baseline snapshot ≥ a {period} old
        {/if}
      </span>
    </div>

    <!-- per-skill gains -->
    <div class="card p-4">
      {#if loadingRates}
        <div class="flex justify-center py-10"><Spinner size={20} /></div>
      {:else if !rates?.hasBaseline}
        <p class="py-6 text-center text-sm text-[var(--color-muted)]">
          Not enough history yet for this period.<br />
          <span class="text-[var(--color-faint)]">
            A {period} of snapshots is needed — the next gains become visible as tracking continues.
          </span>
        </p>
      {:else}
        <div class="mb-4 flex flex-wrap items-baseline gap-3">
          <div>
            <div class="text-2xl font-semibold text-[var(--color-jade)] tabular">
              {signedCompact(rates.overall.gain)}
            </div>
            <div class="text-xs text-[var(--color-muted)]">
              overall xp in this {period} · ≈ {decimal(rates.overall.ratePerDay, 0)} xp/day
            </div>
          </div>
        </div>

        <div class="grid grid-cols-1 gap-2 md:grid-cols-2">
          {#each rates.skills.filter((s) => s.gain > 0).sort((a, b) => b.gain - a.gain) as s (s.key)}
            <div class="flex items-center gap-3 rounded-lg border border-[var(--color-line)] bg-[var(--color-bg-soft)] px-3 py-2">
              <SkillIcon skillKey={s.key} name={s.name} size={26} rounded={7} />
              <div class="w-28 min-w-0">
                <div class="truncate text-xs text-[var(--color-ink)]">{s.name}</div>
                {#if s.rankChange != null && s.rankChange !== 0}
                  {@const up = s.rankChange < 0}
                  <button
                    class="text-[10px] leading-tight {up ? 'text-[var(--color-jade)]' : 'text-[var(--color-rose)]'}"
                    title="Hiscore rank changed during this period"
                  >
                    {up ? '▲' : '▼'} {num(Math.abs(s.rankChange))} rank{Math.abs(s.rankChange) === 1 ? '' : 's'}
                  </button>
                {:else if s.rank != null}
                  <div class="text-[10px] text-[var(--color-faint)]">rank {num(s.rank)}</div>
                {/if}
              </div>
              <div class="flex-1">
                <ProgressBar pctValue={(s.gain / maxGain) * 100} color={skillColor(s.key).accent} height={6} />
              </div>
              <span class="tabular w-20 text-right text-xs font-medium text-[var(--color-jade)]">
                {signedCompact(s.gain)}
              </span>
            </div>
          {/each}
          {#if rates.skills.filter((s) => s.gain > 0).length === 0}
            <p class="col-span-2 py-4 text-center text-xs text-[var(--color-muted)]">
              No XP recorded in this period yet.
            </p>
          {/if}
        </div>
      {/if}
    </div>

    <!-- training pace check: assumed rates.csv rate vs measured pace -->
    <div class="card p-4">
      <div class="mb-1 flex items-center justify-between">
        <h2 class="text-sm font-semibold text-[var(--color-ink)]">Training pace check</h2>
        <span class="text-[11px] text-[var(--color-faint)]">your assumed rates vs actual {period} pace</span>
      </div>
      <p class="mb-3 text-[11px] text-[var(--color-muted)]">
        Compares the XP/hour you configured in Training rates with what you actually gained in this
        {period}. If actual is well below assumed, your estimate is optimistic.
      </p>
      <div class="grid grid-cols-1 gap-2 md:grid-cols-2">
        {#each paceRows as row (row.key)}
          <div class="flex items-center gap-3 rounded-lg border border-[var(--color-line)] bg-[var(--color-bg-soft)] px-3 py-2">
            <SkillIcon skillKey={row.key} name={row.name} size={26} rounded={7} />
            <div class="min-w-0 flex-1">
              <div class="truncate text-xs text-[var(--color-ink)]">{row.name}</div>
              <div class="tabular text-[10px] text-[var(--color-muted)]">
                assumed {compact(row.assumed)} · measured {compact(row.measured)} xp/h
              </div>
            </div>
            <span
              class="tabular rounded-md px-2 py-1 text-[11px] font-semibold"
              style="background:{row.bg};color:{row.fg}"
            >
              {decimal(row.efficiency, 0)}%
            </span>
          </div>
        {/each}
        {#if paceRows.length === 0}
          <p class="col-span-2 py-3 text-center text-xs text-[var(--color-muted)]">
            No pace data yet — set rates in Training rates and record a full {period} of history.
          </p>
        {/if}
      </div>
    </div>

    <!-- history chart -->
    <div class="card p-4">
      <div class="mb-3 flex flex-wrap items-center gap-2">
        <h2 class="text-sm font-semibold text-[var(--color-ink)]">XP over time</h2>
        <div class="flex-1"></div>
        <select class="input !w-auto" bind:value={skill} aria-label="Skill">
          {#each skillOptions as o (o.key)}
            <option value={o.key}>{o.name}</option>
          {/each}
        </select>
        <div class="card inline-flex overflow-hidden p-1">
          {#each RANGES as r (r.days)}
            <button
              class="rounded-lg px-2.5 py-1 text-[11px] font-medium transition-colors
                {rangeDays === r.days
                ? 'bg-[var(--color-panel-2)] text-[var(--color-ink)]'
                : 'text-[var(--color-muted)] hover:text-[var(--color-ink)]'}"
              onclick={() => (rangeDays = r.days)}
            >
              {r.label}
            </button>
          {/each}
        </div>
      </div>

      {#if loadingHistory}
        <div class="flex justify-center py-10"><Spinner size={20} /></div>
      {:else}
        <LineChart
          labels={chartLabels}
          series={[
            {
              label: skill === 'overall' ? 'Overall' : skillOptions.find((o) => o.key === skill)?.name ?? skill,
              color: skillColor(skill).accentHex,
              points: chartPoints,
            },
          ]}
          height={260}
          yFormat={compact}
          tooltipValue={(n) => `+${n.toLocaleString()} xp since previous snapshot`}
          beginAtZero
        />
        <p class="mt-2 text-center text-[11px] text-[var(--color-faint)]">
          XP gained between consecutive snapshots · {history?.points.length ?? 0} data points
        </p>
      {/if}
    </div>
  {/if}
</div>
