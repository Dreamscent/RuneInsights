<script lang="ts">
  import { store } from '../lib/store.svelte';
  import { api, type Rates, type HistoryResponse, type Consistency } from '../lib/api';
  import StatCard from '../components/StatCard.svelte';
  import SkillIcon from '../components/SkillIcon.svelte';
  import Spinner from '../components/Spinner.svelte';
  import LineChart from '../components/LineChart.svelte';
  import ProgressBar from '../components/ProgressBar.svelte';
  import { compact, decimal, duration, num, pct, signedCompact, dateTime } from '../lib/format';

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

  // daily gains for the bar chart: skip the first point (no delta yet)
  const dailyGainLabels = $derived(
    (consistency?.days.filter((d) => d.gain >= 0).slice(-30) ?? []).map((d) =>
      new Date(d.date + 'T00:00:00Z').toLocaleDateString(undefined, { month: 'short', day: 'numeric' }),
    ),
  );
  const dailyGainPoints = $derived(
    consistency?.days.filter((d) => d.gain >= 0).slice(-30).map((d) => d.gain) ?? [],
  );

  const milestonePctAvg = $derived.by(() => {
    const skills = data?.skills ?? [];
    if (skills.length === 0) return 0;
    return skills.reduce((acc, s) => acc + s.next.pct, 0) / skills.length;
  });

  let includeVirtual = $state(true);
  let closestSortByPct = $state(false);

  // Skills closest to gaining their next level, top 5. Toggle to include or
  // hide virtual gains (past what the hiscores display) and to rank by raw
  // XP outstanding or by percent progress into the current level.
  const closestToLevel = $derived.by(() => {
    const skills = data?.skills ?? [];
    return skills
      .filter((s) => s.xpToNext > s.xpIntoLevel && (includeVirtual || !s.nextLevelVirtual))
      .map((s) => ({
        key: s.key,
        name: s.name,
        nextLevel: s.virtualLevel + 1,
        virtual: s.nextLevelVirtual,
        expRequired: s.xpToNext - s.xpIntoLevel,
        pct: (s.xpIntoLevel / s.xpToNext) * 100,
      }))
      .sort((a, b) => (closestSortByPct ? b.pct - a.pct : a.expRequired - b.expRequired))
      .slice(0, 5);
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
        valueSuffix="/ 152"
        accent="#fb7185"
      >
        {#snippet icon()}
          <div class="flex h-10 w-10 items-center justify-center rounded-xl bg-[rgba(251,113,133,.15)] text-xl">⚔️</div>
        {/snippet}
      </StatCard>

      <StatCard
        label="Overall level"
        value={num(data.overall.level)}
        valueSuffix={`/ ${num(data.maxTotalLevel)}`}
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

        <!-- focusing on -->
        <div class="card mb-5 p-4">
          <div class="mb-3 flex items-center justify-between">
            <h2 class="text-sm font-semibold text-[var(--color-ink)]">Focusing on</h2>
            <button
              class="text-[11px] text-[var(--color-gold)] hover:underline"
              onclick={() => (store.view = 'skills')}
            >
              {data.focus.length} pinned · manage
            </button>
          </div>
          {#if data.focus.length === 0}
            <p class="py-4 text-center text-xs text-[var(--color-muted)]">
              Pin skills on the Skills page to follow them here.
            </p>
          {:else}
            <div class="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-4">
              {#each data.focus.slice(0, 8) as fk (fk)}
                {@const sk = (data.skills || []).find((x) => x.key === fk)}
                {@const dayRow = rates.day?.skills.find((x) => x.key === fk)}
                {@const momentum = dayRow?.hasBaseline && dayRow.gain > 0}
                {#if sk}
                  <div class="rounded-lg border border-[var(--color-line)] bg-[var(--color-bg-soft)] p-3">
                    <div class="flex items-center gap-2">
                      <SkillIcon skillKey={sk.key} name={sk.name} size={26} rounded={7} />
                      <div class="min-w-0 leading-tight">
                        <div class="truncate text-xs font-semibold text-[var(--color-ink)]">{sk.name}</div>
                        <div class="text-[10px] text-[var(--color-muted)]">
                          lvl {sk.level}{sk.virtualLevel !== sk.level ? ' (v' + sk.virtualLevel + ')' : ''}
                          · {compact(sk.xp)} xp
                        </div>
                      </div>
                      <span
                        class="ml-auto h-2 w-2 shrink-0 rounded-full"
                        style="background:{momentum ? 'var(--color-jade)' : 'var(--color-line-2)'}"
                        title={momentum ? '+' + num(dayRow.gain) + ' xp gained in the last day' : 'no XP in the last day'}
                      ></span>
                    </div>
                    <div class="mt-2">
                      <ProgressBar
                        pctValue={sk.next.pct}
                        color="linear-gradient(90deg,#34d399,var(--color-gold))"
                        height={5}
                      />
                      <div class="tabular mt-1 flex items-center justify-between text-[10px] text-[var(--color-muted)]">
                        <span>{decimal(sk.next.pct, 0)}% → {sk.next.label}</span>
                        <span class="font-medium {sk.next.etaHours ? 'text-[var(--color-gold-soft)]' : 'text-[var(--color-faint)]'}">
                          {sk.next.etaHours ? '≈ ' + duration(sk.next.etaHours) : 'no estimate'}
                        </span>
                      </div>
                    </div>
                  </div>
                {/if}
              {/each}
            </div>
            {#if data.focus.length > 4}
              <p class="mt-2 text-center text-[11px] text-[var(--color-faint)]">
                +{data.focus.length - 4} more pinned — see the Skills page
              </p>
            {/if}
          {/if}
        </div>


        <div class="grid grid-cols-1 gap-5 lg:grid-cols-5">
      <!-- overall chart -->
      <div class="card p-4 lg:col-span-2">
        <div class="mb-2">
          <h2 class="text-sm font-semibold text-[var(--color-ink)]">Overall XP — last 30 days</h2>
        </div>
        {#if ratesLoading}
          <div class="flex h-96 items-center justify-center"><Spinner size={20} /></div>
        {:else}
          <div class="mb-1">
            <LineChart
              labels={chartLabels}
              series={[{ label: 'Overall', color: '#f5a524', points: chartPoints }]}
              height={180}
              yFormat={compact}
              tooltipValue={(n) => `${n.toLocaleString()} xp`}
            />
          </div>
          <div class="mt-2 border-t border-[var(--color-line)] pt-2">
            <div class="mb-1 flex items-center justify-between">
              <span class="text-sm font-semibold text-[var(--color-ink)]">Daily gains</span>
              <span class="text-[10px] text-[var(--color-faint)]">xp recorded per day</span>
            </div>
            <LineChart
              type="bar"
              labels={dailyGainLabels}
              series={[{ label: 'Daily gain', color: '#34d399', points: dailyGainPoints }]}
              height={140}
              yFormat={compact}
              tooltipValue={(n) => `+${n.toLocaleString()} xp`}
              beginAtZero
            />
          </div>
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
                  stroke-dashoffset={ringC * (1 - milestonePctAvg / 100)}
                  transform="rotate(-90 60 60)"
                  style="transition: stroke-dashoffset 600ms ease"
                ></circle>
              </svg>
              <div class="absolute inset-0 flex flex-col items-center justify-center">
                <span class="tabular text-lg font-semibold">{decimal(milestonePctAvg, 0)}%</span>
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
            </div>
          </div>
        </div>

        <!-- closest to leveling -->
        <div class="card p-4">
          <div class="mb-3 flex items-center justify-between">
            <h2 class="text-sm font-semibold text-[var(--color-ink)]">Closest to leveling</h2>
            <div class="flex items-center gap-2">
              <!-- include/exclude virtual gains -->
              <button
                class="flex cursor-pointer items-center gap-1.5 rounded-full border py-0.5 pr-2 pl-1 text-[10px] font-medium transition-colors {includeVirtual
                  ? 'border-[var(--color-gold-soft)] text-[var(--color-gold-soft)]'
                  : 'border-[var(--color-line)] text-[var(--color-faint)] hover:border-[var(--color-line-2)] hover:text-[var(--color-muted)]'}"
                onclick={() => (includeVirtual = !includeVirtual)}
                aria-pressed={includeVirtual}
                title={includeVirtual
                  ? 'Including skills leveling past the displayed level cap — click to hide them'
                  : 'Virtual levels hidden — click to include them'}
              >
                <span
                  class="relative h-3 w-5 rounded-full transition-colors"
                  style="background:{includeVirtual ? 'var(--color-gold-soft)' : 'var(--color-line-2)'}"
                >
                  <span
                    class="absolute top-[2px] h-2 w-2 rounded-full bg-[var(--color-bg)] transition-all"
                    style="left:{includeVirtual ? '9px' : '1px'}"
                  ></span>
                </span>
                virtual
              </button>
              <!-- ranking metric -->
              <div
                class="flex items-center gap-0.5 rounded-full border border-[var(--color-line)] p-0.5 text-[10px] font-medium"
                role="group"
                aria-label="Ranking metric"
              >
                <button
                  class="cursor-pointer rounded-full px-2 py-[2px] transition-colors {!closestSortByPct
                    ? 'bg-[var(--color-bg-soft)] text-[var(--color-ink)]'
                    : 'text-[var(--color-faint)] hover:text-[var(--color-muted)]'}"
                  onclick={() => (closestSortByPct = false)}
                >
                  xp
                </button>
                <button
                  class="cursor-pointer rounded-full px-2 py-[2px] transition-colors {closestSortByPct
                    ? 'bg-[var(--color-bg-soft)] text-[var(--color-ink)]'
                    : 'text-[var(--color-faint)] hover:text-[var(--color-muted)]'}"
                  onclick={() => (closestSortByPct = true)}
                >
                  %
                </button>
              </div>
            </div>
          </div>
          {#if closestToLevel.length === 0}
            <p class="py-2 text-xs text-[var(--color-muted)]">
              {includeVirtual
                ? 'Every skill is at its XP cap — nothing left to level.'
                : 'Nothing qualifies with virtual levels hidden — toggle them back on.'}
            </p>
          {:else}
            <div class="flex flex-col divide-y divide-[var(--color-line)]">
              {#each closestToLevel as s (s.key)}
                <div class="flex items-center gap-3 py-2">
                  <SkillIcon skillKey={s.key} name={s.name} size={26} rounded={7} />
                  <span class="truncate text-sm text-[var(--color-ink)]">{s.name}</span>
                  {#if s.virtual}
                    <span
                      class="text-[9px] font-semibold uppercase tracking-wide text-[var(--color-gold-soft)]"
                      title="level beyond what the hiscores display"
                      >virtual</span
                    >
                  {/if}
                  {#if closestSortByPct}
                    <span class="tabular ml-auto text-xs font-medium text-[var(--color-ink)]">{pct(s.pct, 1)}</span>
                    <span class="tabular text-[10px] text-[var(--color-faint)]" title="{num(s.expRequired)} xp required">
                      {compact(s.expRequired)} xp
                    </span>
                  {:else}
                    <span class="tabular ml-auto text-xs text-[var(--color-muted)]" title="{num(s.expRequired)} xp required">
                      {compact(s.expRequired)} xp required
                    </span>
                  {/if}
                  <span
                    class="tabular w-12 text-right text-sm font-semibold"
                    style="color:{s.virtual ? 'var(--color-gold-soft)' : 'var(--color-jade)'}"
                    >→ {s.nextLevel}</span
                  >
                </div>
              {/each}
            </div>
          {/if}
        </div>

        <div class="card p-4">
          <h2 class="mb-3 text-sm font-semibold text-[var(--color-ink)]">Tracking Stats</h2>
          <div class="grid grid-cols-2 gap-2 text-[11px] sm:grid-cols-4">
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
