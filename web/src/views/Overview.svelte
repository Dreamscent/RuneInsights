<script lang="ts">
  import { store } from '../lib/store.svelte';
  import { api, type Rates, type HistoryResponse, type Consistency } from '../lib/api';
  import StatCard from '../components/StatCard.svelte';
  import SkillIcon from '../components/SkillIcon.svelte';
  import Spinner from '../components/Spinner.svelte';
  import LineChart from '../components/LineChart.svelte';
  import { compact, decimal, num, pct, signedCompact, dateTime } from '../lib/format';

  const data = $derived(store.skills);

  const PERIODS = ['day', 'week', 'month', 'year'] as const;
  type P = (typeof PERIODS)[number];

  let rates = $state<Record<P, Rates | null>>({ day: null, week: null, month: null, year: null });
  let ratesLoading = $state(true);
  let overallHistory = $state<HistoryResponse | null>(null);
  let topSkills = $state<{ key: string; name: string; gain: number }[]>([]);
  let consistency = $state<Consistency | null>(null);

  async function loadAll(id: number) {
    ratesLoading = true;
    try {
      const [r, h, c] = await Promise.all([
        Promise.all(PERIODS.map((p) => api.rates(id, p))),
        api.history(id, 'overall'),
        api.dailyGains(id),
      ]);
      consistency = c.consistency;
      rates = { day: r[0], week: r[1], month: r[2], year: r[3] };
      overallHistory = h;
      const month = r[2];
      topSkills = (month?.skills ?? [])
        .filter((s) => s.hasBaseline && s.gain > 0)
        .sort((a, b) => b.gain - a.gain)
        .slice(0, 5)
        .map((s) => ({ key: s.key, name: s.name, gain: s.gain }));
    } catch (e) {
      store.notify(e instanceof Error ? e.message : 'Failed to load progress data', 'err');
    } finally {
      ratesLoading = false;
    }
  }

  const loadedId = $state({ current: null as number | null });
  $effect(() => {
    const id = store.selectedId;
    if (id !== null && data && loadedId.current !== id) {
      loadedId.current = id;
      void loadAll(id);
    }
  });

  function gainText(r: Rates | null): string {
    if (!r) return 'collecting…';
    if (r.hasBaseline) return signedCompact(r.overall.gain);
    // Period has no baseline yet — show total XP since tracking began.
    if (r.trackingGain > 0) return signedCompact(r.trackingGain);
    return 'collecting…';
  }

  function periodNote(r: Rates | null): string {
    if (r?.hasBaseline) return `≈ ${decimal(r.overall.ratePerDay, 0)} xp / day`;
    if (r && r.trackingGain > 0) {
      const days = Math.max(1, Math.floor(r.trackingDays));
      return `since tracking began · ${days} ${days === 1 ? 'day' : 'days'}`;
    }
    return 'collecting data…';
  }

  const chartLabels = $derived(
    (overallHistory?.points ?? []).map((p) =>
      new Date(p.at).toLocaleDateString(undefined, { month: 'short', day: 'numeric' }),
    ),
  );
  const chartPoints = $derived((overallHistory?.points ?? []).map((p) => p.xp));

  const milestoneStats = $derived.by(() => {
    const skills = data?.skills ?? [];
    if (skills.length === 0) return { pctAvg: 0, complete: 0, count: 0, closest: [] };
    const pctAvg = skills.reduce((acc, s) => acc + s.next.pct, 0) / skills.length;
    const complete = skills.filter((s) => s.next.remaining === 0).length;
    const closest = skills
      .filter((s) => s.next && s.next.remaining > 0)
      .sort((a, b) => b.next.pct - a.next.pct)
      .slice(0, 3);
    return { pctAvg, complete, count: skills.length, closest };
  });

  const maxedCount = $derived((data?.skills ?? []).filter((s) => s.maxed).length);
  const topSkillByXP = $derived(
    data?.skills.reduce((best, s) => (s.xp > (best?.xp ?? -1) ? s : best), data.skills[0]) ?? null,
  );

  const ringR = 54;
  const ringC = 2 * Math.PI * ringR;

  // count color: red (0) -> green (total), interpolated on the hue wheel
  function progressColor(count: number, total: number): string {
    const frac = total > 0 ? Math.max(0, Math.min(1, count / total)) : 0;
    const hue = Math.round(frac * 120);
    return `hsl(${hue} 78% 60%)`;
  }
</script>

<div class="flex flex-col gap-5">
  {#if !data}
    <div class="flex justify-center py-16"><Spinner /></div>
  {:else}
    <div class="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
      <StatCard
        label="Overall XP"
        value={compact(data.overall.xp)}
        sub={data.overall.rank ? `Global rank #${num(data.overall.rank)}` : 'Unranked'}
        accent="var(--color-gold)"
      >
        {#snippet icon()}
          <SkillIcon skillKey="overall" size={40} rounded={12} />
        {/snippet}
      </StatCard>

      <StatCard
        label="Combat level"
        value={String(data.combatLevel)}
        sub="max 152"
        accent="#fb7185"
        footnote="Computed from combat stats (RS3 formula)"
      >
        {#snippet icon()}
          <div class="flex h-10 w-10 items-center justify-center rounded-xl bg-[rgba(251,113,133,.15)] text-xl">⚔️</div>
        {/snippet}
      </StatCard>

      <StatCard
        label="Overall level"
        value={num(data.overall.level)}
        sub={`${maxedCount} skill${maxedCount === 1 ? '' : 's'} maxed`}
        accent="#38bdf8"
      >
        {#snippet icon()}
          <div class="flex h-10 w-10 items-center justify-center rounded-xl bg-[rgba(56,189,248,.15)] text-xl">📈</div>
        {/snippet}
      </StatCard>

      <StatCard
        label="Top skill (XP)"
        value={topSkillByXP ? compact(topSkillByXP.xp) : '—'}
        sub={topSkillByXP?.name ? `${topSkillByXP.name} · level ${topSkillByXP.level}` : undefined}
        accent="#a78bfa"
      >
        {#snippet icon()}
          {#if topSkillByXP}
            <SkillIcon skillKey={topSkillByXP.key} name={topSkillByXP.name} size={40} rounded={12} />
          {/if}
        {/snippet}
      </StatCard>
    </div>

    <!-- gain summary -->
    <div class="card p-4">
      <div class="mb-3 flex items-center justify-between">
        <h2 class="text-sm font-semibold text-[var(--color-ink)]">XP gained</h2>
        <span class="text-[11px] text-[var(--color-faint)]">overall · from snapshot history</span>
      </div>
      <div class="grid grid-cols-2 gap-3 sm:grid-cols-4">
        {#each PERIODS as p (p)}
          {@const r = rates[p]}
          <div class="rounded-xl border border-[var(--color-line)] bg-[var(--color-bg-soft)] p-3">
            <div class="text-[10px] font-semibold tracking-widest text-[var(--color-faint)] uppercase">
              {p}
            </div>
            <div
              class="tabular mt-1 text-lg font-semibold {r?.hasBaseline
                ? 'text-[var(--color-jade)]'
                : 'text-[var(--color-muted)]'}"
            >
              {gainText(r)}
            </div>
            <div class="text-[11px] text-[var(--color-muted)]">{periodNote(r)}</div>
          </div>
        {/each}
      </div>
    </div>

    <div class="grid grid-cols-1 gap-5 lg:grid-cols-5">
      <!-- overall chart -->
      <div class="card p-4 lg:col-span-2">
        <div class="mb-2">
          <h2 class="text-sm font-semibold text-[var(--color-ink)]">Overall XP — last 30 days</h2>
        </div>
        {#if ratesLoading}
          <div class="flex h-52 items-center justify-center"><Spinner size={20} /></div>
        {:else}
          <LineChart
            labels={chartLabels}
            series={[{ label: 'Overall', color: '#f5a524', points: chartPoints }]}
            height={220}
            yFormat={compact}
            tooltipValue={(n) => `${n.toLocaleString()} xp`}
          />
        {/if}
      </div>

      <!-- milestones + movers -->
      <div class="flex flex-col gap-5 lg:col-span-3">
        <div class="card p-4">
          <h2 class="mb-3 text-sm font-semibold text-[var(--color-ink)]">Milestone progress</h2>
          <div class="flex items-center gap-4">
            <div class="relative shrink-0">
              <svg width="120" height="120" viewBox="0 0 120 120">
                <circle cx="60" cy="60" r={ringR} fill="none" stroke="rgba(255,255,255,0.07)" stroke-width="9" />
                <circle
                  cx="60"
                  cy="60"
                  r={ringR}
                  fill="none"
                  stroke="var(--color-jade)"
                  stroke-width="9"
                  stroke-linecap="round"
                  stroke-dasharray={ringC}
                  stroke-dashoffset={ringC * (1 - milestoneStats.pctAvg / 100)}
                  transform="rotate(-90 60 60)"
                  style="transition: stroke-dashoffset 600ms ease"
                ></circle>
              </svg>
              <div class="absolute inset-0 flex flex-col items-center justify-center">
                <span class="tabular text-lg font-semibold">{decimal(milestoneStats.pctAvg, 0)}%</span>
                <span class="text-[10px] text-[var(--color-faint)]">toward 200m</span>
              </div>
            </div>
            <div class="min-w-0 flex-1">
              <div class="grid grid-cols-2 gap-2">
                <div class="rounded-lg border border-[var(--color-line)] bg-[var(--color-bg-soft)] p-2 text-center">
                  <div class="tabular text-base font-semibold" style="color:{progressColor(data.milestoneCounts.skillsAt99, data.milestoneCounts.total)}">
                    {data.milestoneCounts.skillsAt99}<span class="text-[11px] font-medium text-[var(--color-jade)]">/{data.milestoneCounts.total}</span>
                  </div>
                  <div class="text-[10px] leading-tight text-[var(--color-faint)]">
                    skills at 99<span class="block text-[9px]">13.0M xp</span>
                  </div>
                </div>
                <div class="rounded-lg border border-[var(--color-line)] bg-[var(--color-bg-soft)] p-2 text-center">
                  <div class="tabular text-base font-semibold" style="color:{progressColor(data.milestoneCounts.skillsAt110, data.milestoneCounts.total)}">
                    {data.milestoneCounts.skillsAt110}<span class="text-[11px] font-medium text-[var(--color-jade)]">/{data.milestoneCounts.total}</span>
                  </div>
                  <div class="text-[10px] leading-tight text-[var(--color-faint)]">
                    skills at 110<span class="block text-[9px]">38.7M xp</span>
                  </div>
                </div>
                <div class="rounded-lg border border-[var(--color-line)] bg-[var(--color-bg-soft)] p-2 text-center">
                  <div class="tabular text-base font-semibold" style="color:{progressColor(data.milestoneCounts.skillsAt120, data.milestoneCounts.total)}">
                    {data.milestoneCounts.skillsAt120}<span class="text-[11px] font-medium text-[var(--color-jade)]">/{data.milestoneCounts.total}</span>
                  </div>
                  <div class="text-[10px] leading-tight text-[var(--color-faint)]">
                    skills at 120<span class="block text-[9px]">104.3M xp</span>
                  </div>
                </div>
                <div class="rounded-lg border border-[var(--color-line)] bg-[var(--color-bg-soft)] p-2 text-center">
                  <div class="tabular text-base font-semibold" style="color:{progressColor(data.milestoneCounts.skillsAtLevelCap, data.milestoneCounts.total)}">
                    {data.milestoneCounts.skillsAtLevelCap}<span class="text-[11px] font-medium text-[var(--color-jade)]">/{data.milestoneCounts.total}</span>
                  </div>
                  <div class="text-[10px] leading-tight text-[var(--color-faint)]">at level cap</div>
                  <div class="mt-1 leading-tight">
                    <span class="text-[10px] tracking-wide text-[var(--color-faint)] uppercase">Total level</span>
                    <div class="tabular text-xs font-semibold text-[var(--color-ink)]">
                      {num(data.overall.level)}<span class="font-normal text-[var(--color-faint)]">/{num(data.maxTotalLevel)}</span>
                    </div>
                  </div>
                </div>
                <div class="rounded-lg border border-[var(--color-line)] bg-[var(--color-bg-soft)] p-2 text-center">
                  <div class="tabular text-base font-semibold" style="color:{progressColor(data.milestoneCounts.skillsAt200m, data.milestoneCounts.total)}">
                    {data.milestoneCounts.skillsAt200m}<span class="text-[11px] font-medium text-[var(--color-jade)]">/{data.milestoneCounts.total}</span>
                  </div>
                  <div class="text-[10px] leading-tight text-[var(--color-faint)]">
                    skills at 200m<span class="block text-[9px]">XP cap</span>
                  </div>
                </div>
              </div>
              {#if milestoneStats.closest.length > 0}
                <div class="mt-2 space-y-1">
                  {#each milestoneStats.closest as s (s.key)}
                    <div class="flex items-center gap-1.5">
                      <SkillIcon skillKey={s.key} name={s.name} size={16} rounded={4} dim />
                      <span class="truncate text-[var(--color-muted)]">{s.name}</span>
                      <span class="tabular ml-auto text-[var(--color-jade)]">
                        {pct(Math.min(100, s.next.pct), 0)}
                      </span>
                    </div>
                  {/each}
                </div>
              {/if}
            </div>
          </div>
        </div>

        <div class="card p-4">
          <h2 class="mb-3 text-sm font-semibold text-[var(--color-ink)]">Consistency</h2>
          {#if consistency}
            <div class="flex items-end gap-1" style="height:44px" aria-hidden="true">
              {#each consistency.days.slice(-14) as d, i (d.date)}
                {@const maxG = Math.max(1, ...consistency!.days.slice(-14).map((x) => x.gain))}
                {@const h = d.gain < 0 ? 4 : Math.max(3, Math.round((d.gain / maxG) * 38))}
                <div
                  class="flex-1 rounded-t transition-all"
                  style="height:{h}px;background:{d.gain > 0 ? 'var(--color-jade)' : 'rgba(255,255,255,.08)'}"
                  title="{d.date}: {d.gain < 0 ? 'first snapshot' : signedCompact(d.gain) + ' xp'}"
                ></div>
              {/each}
            </div>
          {/if}
          <div class="mt-3 grid grid-cols-2 gap-2 text-[11px] sm:grid-cols-4">
            <div class="rounded-lg border border-[var(--color-line)] bg-[var(--color-bg-soft)] p-2 text-center">
              <div class="tabular text-sm font-semibold text-[var(--color-ink)]">{num(consistency?.activeDays ?? 0)}</div>
              <div class="text-[10px] text-[var(--color-faint)]">active days</div>
            </div>
            <div class="rounded-lg border border-[var(--color-line)] bg-[var(--color-bg-soft)] p-2 text-center">
              <div class="tabular text-sm font-semibold text-[var(--color-jade)]">{signedCompact(consistency?.totalGained ?? 0)}</div>
              <div class="text-[10px] text-[var(--color-faint)]">xp gained</div>
            </div>
            <div class="rounded-lg border border-[var(--color-line)] bg-[var(--color-bg-soft)] p-2 text-center">
              <div class="tabular text-sm font-semibold text-[var(--color-gold-soft)]">{signedCompact(consistency?.bestGain ?? 0)}</div>
              <div class="text-[10px] text-[var(--color-faint)]">best day{consistency?.bestDate ? ` (${new Date(consistency.bestDate + 'T00:00:00Z').toLocaleDateString(undefined, { month: 'short', day: 'numeric' })})` : ''}</div>
            </div>
            <div class="rounded-lg border border-[var(--color-line)] bg-[var(--color-bg-soft)] p-2 text-center">
              <div class="tabular text-sm font-semibold text-[var(--color-sky)]">{num(consistency?.streak ?? 0)}</div>
              <div class="text-[10px] text-[var(--color-faint)]">day streak</div>
            </div>
          </div>
          {#if !consistency || consistency.days.length < 2}
            <p class="mt-2 text-[11px] text-[var(--color-muted)]">Streaks and best days appear as snapshots accumulate.</p>
          {/if}
        </div>

        <div class="card p-4">
          <h2 class="mb-3 text-sm font-semibold text-[var(--color-ink)]">Top movers (30d)</h2>
          {#if topSkills.length === 0}
            <p class="text-xs text-[var(--color-muted)]">
              Movers appear once two snapshots exist and you log XP.
            </p>
          {:else}
            <div class="flex flex-col divide-y divide-[var(--color-line)]">
              {#each topSkills as s (s.key)}
                <div class="flex items-center gap-3 py-2">
                  <SkillIcon skillKey={s.key} name={s.name} size={26} rounded={7} />
                  <span class="truncate text-sm text-[var(--color-ink)]">{s.name}</span>
                  <span class="tabular ml-auto text-sm font-medium text-[var(--color-jade)]">
                    {signedCompact(s.gain)}
                  </span>
                </div>
              {/each}
            </div>
          {/if}
        </div>
      </div>
    </div>

    <p class="text-center text-[11px] text-[var(--color-faint)]">
      Last snapshot {dateTime(data.snapshotAt)} · collected automatically every {store.selected?.intervalMin}
      min
    </p>
  {/if}
</div>
