<script lang="ts">
  import { store } from '../lib/store.svelte';
  import { compact, decimal, duration, num, signedCompact, timeAgo } from '../lib/format';
  import SkillIcon from '../components/SkillIcon.svelte';
  import ProgressBar from '../components/ProgressBar.svelte';
  import Modal from '../components/Modal.svelte';
  import LineChart from '../components/LineChart.svelte';
  import EmptyState from '../components/EmptyState.svelte';
  import Spinner from '../components/Spinner.svelte';
  import { api, type SkillView, type HistoryResponse, type RateEntry, type Rates } from '../lib/api';
  import { skillColor } from '../lib/skills';

  const data = $derived(store.skills);

  type SortKey = 'name' | 'level' | 'xp' | 'rank' | 'remaining' | 'rate';
  type Dir = 'asc' | 'desc';
  let sortKey = $state<SortKey>('name');
  let sortDir = $state<Dir>('asc');
  let query = $state('');

  function toggleSort(k: SortKey) {
    if (sortKey === k) {
      sortDir = sortDir === 'asc' ? 'desc' : 'asc';
    } else {
      sortKey = k;
      sortDir = k === 'name' ? 'asc' : 'desc';
    }
  }

  const filtered = $derived.by(() => {
    const list = data?.skills ?? [];
    const q = query.trim().toLowerCase();
    const base = q ? list.filter((s) => s.name.toLowerCase().includes(q)) : [...list];
    const dir = sortDir === 'asc' ? 1 : -1;
    return base.sort((a, b) => {
      switch (sortKey) {
        case 'name':
          return a.name.localeCompare(b.name) * dir;
        case 'level':
          return (b.level - a.level || b.xp - a.xp) * (dir === 1 ? -1 : 1);
        case 'xp':
          return (a.xp - b.xp) * dir;
        case 'rank': {
          const ra = a.rank ?? Number.MAX_SAFE_INTEGER;
          const rb = b.rank ?? Number.MAX_SAFE_INTEGER;
          return (ra - rb) * dir;
        }
        case 'remaining': {
          const ra = a.next?.remaining ?? -1;
          const rb = b.next?.remaining ?? -1;
          return (ra - rb) * dir;
        }
        case 'rate':
          return ((gainValues.get(a.key) ?? -1) - (gainValues.get(b.key) ?? -1)) * dir;
        default:
          return 0;
      }
    });
  });

  // selected skill detail (history drawer)
  let selected = $state<SkillView | null>(null);

  // gains for the selected window (1/7/30 days), per skill
  const GAIN_WINDOWS = ['day', 'week', 'month'] as const;
  type GainWindow = (typeof GAIN_WINDOWS)[number];
  let gainWindow = $state<GainWindow>('day');
  let ratesByWindow = $state<Partial<Record<GainWindow, Rates | null>>>({});
  const gainValues = $derived.by(() => {
    const m = new Map<string, number | null>();
    for (const sg of ratesByWindow[gainWindow]?.skills ?? []) {
      m.set(sg.key, sg.hasBaseline ? sg.gain : null);
    }
    return m;
  });

  const fallbackGain = $derived.by(() => {
    // per-skill totals since tracking, shown when the selected window has no baseline
    const r = ratesByWindow[gainWindow];
    if (!r || r.hasBaseline) return { map: new Map<string, number>(), active: false, days: r?.trackingDays ?? 0 };
    const m = new Map<string, number>();
    for (const sg of r?.skills ?? []) {
      if (sg.trackingGain > 0) m.set(sg.key, sg.trackingGain);
    }
    return { map: m, active: m.size > 0, days: r?.trackingDays ?? 0 };
  });

  $effect(() => {
    const id = store.selectedId;
    if (id == null) return;
    const cur = gainWindow;
    if (ratesByWindow[cur] !== undefined && ratesByWindow[cur] !== null) return; // cached
    void api
      .rates(id, cur)
      .then((r) => {
        ratesByWindow = { ...ratesByWindow, [cur]: r };
      })
      .catch(() => {
        ratesByWindow = { ...ratesByWindow, [cur]: null };
      });
  });

  function setGainWindow(w: GainWindow) {
    gainWindow = w; // the effect below refetches when the window changes
  }
  let history = $state<HistoryResponse | null>(null);
  let historyLoading = $state(false);

  $effect(() => {
    if (store.selectedId === null) selected = null;
  });

  async function openSkill(s: SkillView) {
    selected = s;
    methodChoices = [];
    methodIdx = 0;
    calcRate = s.xpPerHour > 0 ? String(s.xpPerHour) : '';
    calcMethod = s.method;
    history = null;
    void loadMethodChoices(s.key);
    historyLoading = true;
    try {
      history = await api.history(store.selectedId!, s.key);
    } catch {
      history = null;
    } finally {
      historyLoading = false;
    }
  }

  function errMessage(e: unknown): string {
    return e instanceof Error ? e.message : 'Something went wrong';
  }

  // ---- XP calculator (drawer) ----
  let calcRate = $state('');
  let calcMethod = $state('');
  let calcSaving = $state(false);
  let methodChoices = $state<RateEntry[]>([]);
  let methodIdx = $state(0);

  async function loadMethodChoices(key: string) {
    try {
      const resp = await api.skillRates();
      const row = resp.skills.find((s) => s.key === key);
      methodChoices = row?.methods ?? [];
      methodIdx = 0;
      if (methodChoices.length > 1) {
        calcRate = methodChoices[0].perHour > 0 ? String(methodChoices[0].perHour) : '';
        calcMethod = methodChoices[0].method;
      }
    } catch {
      methodChoices = [];
    }
  }

  function pickMethod(i: number) {
    methodIdx = i;
    if (methodChoices[i]) {
      calcRate = methodChoices[i].perHour > 0 ? String(methodChoices[i].perHour) : '';
      calcMethod = methodChoices[i].method;
    }
  }

  const calcEffective = $derived.by(() => {
    const typed = Number(String(calcRate ?? '').replace(/[,\s]/g, ''));
    if (Number.isFinite(typed) && typed > 0) return typed;
    return selected?.xpPerHour ?? 0;
  });

  async function saveRate() {
    if (!selected || calcSaving) return;
    const n = Number(String(calcRate ?? '').replace(/[,\s]/g, ''));
    calcSaving = true;
    try {
      const list = methodChoices.length > 0
        ? methodChoices.map((m) => ({ ...m }))
        : [{ perHour: 0, method: '' }];
      if (list[methodIdx]) {
        list[methodIdx] = {
          perHour: Number.isFinite(n) && n > 0 ? Math.round(n) : 0,
          method: calcMethod.trim(),
        };
      }
      await api.setSkillRate(selected.key, list);
      await store.reloadSkills();
      const k = selected.key;
      const fresh = store.skills?.skills.find((s) => s.key === k);
      if (fresh) selected = fresh;
      await loadMethodChoices(k);
      calcRate = list[methodIdx]?.perHour > 0 ? String(list[methodIdx].perHour) : '';
      calcMethod = list[methodIdx]?.method ?? calcMethod;
      store.notify('Rate saved to data/skill_rates.csv', 'ok');
    } catch (err) {
      store.notify(errMessage(err), 'err');
    } finally {
      calcSaving = false;
    }
  }

  const arrowChar = (k: SortKey) => (sortKey === k ? (sortDir === 'asc' ? '▲' : '▼') : '');

  const chartColor = $derived(selected ? skillColor(selected.key).accentHex : '#f5a524');
  const chartLabels = $derived(
    (history?.points ?? []).map((p) =>
      new Date(p.at).toLocaleString(undefined, { month: 'short', day: 'numeric', hour: '2-digit' }),
    ),
  );
  const chartPoints = $derived((history?.points ?? []).map((p) => p.xp));
</script>

<div class="flex flex-col gap-4">
  {#if data && data.skills.length > 0}
    <div class="flex flex-wrap items-center gap-2">
      <input
        class="input max-w-56"
        placeholder="Filter skills…"
        bind:value={query}
        aria-label="Filter skills"
      />
      <div class="card inline-flex overflow-hidden p-1" role="group" aria-label="Gain window">
        {#each GAIN_WINDOWS as w (w)}
          <button
            class="rounded-lg px-3 py-1.5 text-xs font-medium transition-colors
              {gainWindow === w
              ? 'bg-[var(--color-panel-2)] text-[var(--color-gold-soft)] shadow-[inset_0_0_0_1px_var(--color-line-2)]'
              : 'text-[var(--color-muted)] hover:text-[var(--color-ink)]'}"
            onclick={() => setGainWindow(w)}
          >
            {w === 'day' ? '1 day' : w === 'week' ? '7 days' : '30 days'}
          </button>
        {/each}
      </div>
      <span class="text-xs text-[var(--color-faint)]">
        Click a skill for its history and automatic next milestone
      </span>
    </div>

    <div class="card overflow-hidden">
      <div class="overflow-x-auto">
        <table class="w-full text-sm">
          <thead>
            <tr class="border-b border-[var(--color-line)] text-left text-[11px] tracking-wider text-[var(--color-faint)] uppercase">
              <th class="cursor-pointer px-4 py-2.5 font-medium select-none" onclick={() => toggleSort('name')}>
                Skill {arrowChar('name')}
              </th>
              <th class="cursor-pointer px-3 py-2.5 text-right font-medium select-none" onclick={() => toggleSort('level')}>
                Level {arrowChar('level')}
              </th>
              <th class="cursor-pointer px-3 py-2.5 text-right font-medium select-none" onclick={() => toggleSort('xp')}>
                Experience {arrowChar('xp')}
              </th>
              <th class="cursor-pointer hidden px-3 py-2.5 text-right font-medium select-none md:table-cell" onclick={() => toggleSort('rate')}>
                Gain {gainWindow === 'day' ? '1d' : gainWindow === 'week' ? '1w' : '30d'} {arrowChar('rate')}
              </th>
              <th class="px-3 py-2.5 font-medium">Next level</th>
              <th class="cursor-pointer px-3 py-2.5 font-medium select-none" onclick={() => toggleSort('remaining')}>
                Next milestone {arrowChar('remaining')}
              </th>
              <th class="cursor-pointer hidden px-3 py-2.5 text-right font-medium select-none lg:table-cell" onclick={() => toggleSort('rank')}>
                Rank {arrowChar('rank')}
              </th>
            </tr>
          </thead>
          <tbody class="divide-row">
            {#each filtered as s (s.key)}
              {@const color = skillColor(s.key)}
              <tr
                class="cursor-pointer transition-colors hover:bg-[var(--color-panel-2)]"
                onclick={() => void openSkill(s)}
              >
                <td class="px-4 py-2.5">
                  <div class="flex items-center gap-2.5">
                    <SkillIcon skillKey={s.key} name={s.name} size={30} rounded={8} />
                    <div class="leading-tight">
                      <div class="font-medium text-[var(--color-ink)]">{s.name}</div>
                      {#if s.elite}
                        <div class="text-[10px] text-[var(--color-faint)]">elite skill</div>
                      {/if}
                    </div>
                  </div>
                </td>

                <td class="tabular px-3 py-2.5 text-right">
                  <span class="font-semibold text-[var(--color-ink)]">{s.level}</span>
                  {#if s.virtualLevel > s.level && s.virtualLevel <= 126}
                    <span class="text-xs text-[var(--color-gold)]" title="Virtual level from XP">v{s.virtualLevel}</span>
                  {/if}
                </td>

                <td class="tabular px-3 py-2.5 text-right text-[var(--color-ink)]" title={s.xp.toLocaleString()}>
                  {compact(s.xp)}
                </td>

                <td class="tabular hidden px-3 py-2.5 text-right md:table-cell">
                  {#if gainValues.get(s.key) === null}
                    {#if fallbackGain.active && fallbackGain.map.get(s.key)}
                      <span class="text-[var(--color-sky)]" title="No full {gainWindow} of history yet — showing total XP gained since tracking began">
                        {signedCompact(fallbackGain.map.get(s.key)!)}
                      </span>
                    {:else}
                      <span class="text-[var(--color-faint)]" title="Need a baseline snapshot ≥ {gainWindow === 'day' ? '1 day' : gainWindow === 'week' ? '1 week' : '30 days'} old">—</span>
                    {/if}
                  {:else}
                    <span class="text-[var(--color-jade)]">{signedCompact(gainValues.get(s.key))}</span>
                  {/if}
                </td>

                <td class="min-w-56 px-3 py-2.5">
                  {#if s.maxed}
                    <span class="text-xs font-medium text-[var(--color-gold)]">MAXED</span>
                  {:else if s.elite}
                    <span class="text-xs text-[var(--color-faint)]">different XP curve</span>
                  {:else}
                    {@const filled = Math.round((s.xpIntoLevel / Math.max(1, s.xpIntoLevel + s.xpToNext)) * 100)}
                    <ProgressBar pctValue={filled} color={color.accent} height={6} />
                    <div class="tabular mt-1 text-[11px] text-[var(--color-muted)]">
                      {compact(s.xpToNext)} xp to level {s.level + 1}
                      {#if s.nextLevelEtaHours != null}<span class="text-[var(--color-gold)]"> · ≈ {duration(s.nextLevelEtaHours)}</span>{/if}
                    </div>
                  {/if}
                </td>
<td class="min-w-56 px-3 py-2.5">
                  {#if s.next.remaining === 0}
                    <span class="text-xs font-medium text-[var(--color-gold)]">{s.next.label} reached</span>
                  {:else}
                    <div class="tabular text-[11px] text-[var(--color-muted)]">
                      {s.next.label} · {compact(s.next.remaining)} left
                      {#if s.next.etaHours}<span class="text-[var(--color-gold)]"> · ≈ {duration(s.next.etaHours)}</span>{/if}
                    </div>
                    <ProgressBar pctValue={s.next.pct} color="linear-gradient(90deg,#34d399,var(--color-gold))" height={6} />
                  {/if}
                </td>

                <td class="tabular hidden px-3 py-2.5 text-right text-[var(--color-muted)] lg:table-cell">
                  {#if s.rank}
                    {s.rank.toLocaleString()}
                  {:else}
                    <span class="text-[var(--color-faint)]">—</span>
                  {/if}
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    </div>

    <p class="text-center text-[11px] text-[var(--color-faint)]">
      Snapshot as of {timeAgo(data.snapshotAt)} · the gain column shows XP recorded in the selected {gainWindow === 'day' ? 'day' : gainWindow === 'week' ? 'week' : '30 days'}
      {#if fallbackGain.active && !ratesByWindow[gainWindow]?.hasBaseline}
        · overall gain is the total since tracking began ({Math.max(1, Math.floor(fallbackGain.days))} {Math.floor(fallbackGain.days) <= 1 ? 'day' : 'days'} of data)
      {/if}
      · each skill drawer shows the trailing 30-day pace
      {#if !data.ratesAvailable}
        · no training-rate estimates — fill in data/skill_rates.csv
      {/if}
      {#if !data.overall || data.overall.ratePerDay === 0}
        (collecting data — a second snapshot is needed first)
      {/if}
    </p>
  {:else if !store.skillsLoading}
    <EmptyState title="No skill data yet" message="Add a player or refresh to capture the first snapshot." />
  {/if}
</div>

{#if selected}
  <Modal
    open={!!selected}
    onclose={() => (selected = null)}
    title={selected.name}
    subtitle={selected.elite ? 'Elite skill — uses its own XP curve (handled exactly, virtual cap 150).' : undefined}
    width="34rem"
  >
    <div class="flex flex-col gap-5">
      <div class="grid grid-cols-3 gap-2 text-center">
        <div class="rounded-lg border border-[var(--color-line)] bg-[var(--color-bg-soft)] p-2.5">
          <div class="text-[10px] tracking-wider text-[var(--color-faint)] uppercase">Level</div>
          <div class="tabular text-lg font-semibold">
            {selected.level}
            {#if selected.virtualLevel !== selected.level}<span class="text-xs text-[var(--color-gold)]">v{selected.virtualLevel}</span>{/if}
          </div>
        </div>
        <div class="rounded-lg border border-[var(--color-line)] bg-[var(--color-bg-soft)] p-2.5">
          <div class="text-[10px] tracking-wider text-[var(--color-faint)] uppercase">XP</div>
          <div class="tabular text-lg font-semibold">{compact(selected.xp)}</div>
        </div>
        <div class="rounded-lg border border-[var(--color-line)] bg-[var(--color-bg-soft)] p-2.5">
          <div class="text-[10px] tracking-wider text-[var(--color-faint)] uppercase">Rank</div>
          <div class="tabular text-lg font-semibold">{selected.rank ?? '—'}</div>
        </div>
      </div>

      {#if !selected.maxed}
        <div>
          <div class="mb-1.5 flex items-center justify-between text-xs text-[var(--color-muted)]">
            <span>Progress to level {selected.level + 1}</span>
            <span class="tabular">{compact(selected.xpIntoLevel)} / {compact(selected.xpIntoLevel + selected.xpToNext)}</span>
          </div>
          <ProgressBar
            pctValue={(selected.xpIntoLevel / Math.max(1, selected.xpIntoLevel + selected.xpToNext)) * 100}
            color={skillColor(selected.key).accent}
            height={10}
          />
        </div>
      {/if}

      <div>
        <div class="mb-2 flex items-center justify-between">
          <span class="text-xs font-medium text-[var(--color-muted)]">XP history (last 30 days)</span>
          {#if selected.ratePerDay > 0.5}
            <span class="chip"><span class="text-[var(--color-jade)]">{signedCompact(selected.ratePerDay)}</span> xp / day (30d avg)</span>
          {/if}
        </div>
        <div class="card p-2">
          {#if historyLoading}
            <div class="flex h-40 items-center justify-center">
              <Spinner size={20} />
            </div>
          {:else}
            <LineChart
              labels={chartLabels}
              series={[{ label: selected.name, color: chartColor, points: chartPoints }]}
              height={190}
              yFormat={compact}
              tooltipValue={(n) => `${n.toLocaleString()} xp`}
              beginAtZero
            />
          {/if}
        </div>
      </div>

      <div class="rounded-lg border border-[var(--color-line)] bg-[var(--color-bg-soft)] p-3">
        <div class="mb-2 flex items-center justify-between text-xs font-medium text-[var(--color-muted)]">
          <span>Next milestone: {selected.next.label}</span>
          <span class="tabular">{decimal(selected.next.pct)}%</span>
        </div>
        <ProgressBar pctValue={selected.next.pct} color="linear-gradient(90deg,#34d399,var(--color-gold))" height={8} />
        <div class="tabular mt-2 flex items-center justify-between text-[11px] text-[var(--color-muted)]">
          <span>
            {compact(selected.xp)} / {compact(selected.next.xp)}
            {#if selected.next.remaining > 0}· {compact(selected.next.remaining)} xp left{/if}
          </span>
          {#if selected.next.etaHours}
            <span>ETA ~{duration(selected.next.etaHours)}</span>
          {/if}
        </div>
        <p class="mt-2 text-[10px] text-[var(--color-faint)]">
          Milestones follow the game tiers: level 99 · 110 · 120{selected.elite ? ' · 150 (elite)' : ''} · 200m xp
        </p>
      </div>

      <!-- XP calculator -->
      <div class="rounded-lg border border-[var(--color-line)] bg-[var(--color-bg-soft)] p-3">
        <div class="mb-2.5 flex items-center justify-between">
          <span class="text-xs font-medium text-[var(--color-muted)]">XP calculator</span>
          <span class="chip !py-0 text-[10px]">live</span>
        </div>

        <div class="flex flex-wrap items-end gap-2">
          <label class="min-w-32 flex-1 text-[10px] tracking-wider text-[var(--color-faint)] uppercase">
            XP / hour
            <input
              class="input tabular mt-1"
              inputmode="numeric"
              bind:value={calcRate}
              placeholder="e.g. 350000"
            />
          </label>
          <label class="min-w-36 flex-1 text-[10px] tracking-wider text-[var(--color-faint)] uppercase">
            Training method
            <input class="input mt-1" bind:value={calcMethod} placeholder="e.g. Afk Croesus Front" />
          </label>
          <button
            class="btn btn-gold"
            disabled={calcSaving}
            onclick={() => void saveRate()}
          >
            {calcSaving ? 'Saving…' : 'Save rate'}
          </button>
        </div>

        {#if methodChoices.length > 1}
          <div class="mt-3 flex flex-col gap-1">
            <span class="text-[10px] font-semibold tracking-widest text-[var(--color-faint)] uppercase">
              Available methods ({methodChoices.length}) — first is the default
            </span>
            {#each methodChoices as m, i (i)}
              {@const t = m.perHour > 0
                ? (i === 0
                  ? `to level ${selected.level + 1}: ≈ ${duration((selected.xpToNext - selected.xpIntoLevel) / m.perHour)} · to ${selected.next.label}: ≈ ${duration(selected.next.remaining / m.perHour)}`
                  : `to ${selected.next.label}: ≈ ${duration(selected.next.remaining / m.perHour)}`)
                : 'no estimate'}
              <button
                class="flex items-center gap-2 rounded-lg border px-2.5 py-1.5 text-left text-[11px] transition-colors
                  {methodIdx === i
                  ? 'border-[var(--color-gold)] bg-[rgba(245,165,36,.08)]'
                  : 'border-[var(--color-line)] hover:border-[var(--color-line-2)]'}"
                onclick={() => pickMethod(i)}
              >
                <span class="chip !py-0 text-[10px] {i === 0 ? 'text-[var(--color-gold)]' : ''}">
                  {i === 0 ? 'default' : `#${i + 1}`}
                </span>
                <span class="truncate text-[var(--color-ink)]">{m.method || '— unnamed —'}</span>
                <span class="tabular ml-auto text-[var(--color-muted)]">{m.perHour > 0 ? `${compact(m.perHour)} xp/h` : '—'}</span>
              </button>
            {/each}
          </div>
        {/if}

        {#if calcEffective > 0}
          <div class="mt-3 rounded-lg border border-[var(--color-line)] bg-[var(--color-panel)] p-2.5 text-[11px]">
            <div class="tabular flex items-center justify-between">
              <span class="text-[var(--color-muted)]">Time to level {selected.level + 1}</span>
              <span class="font-medium text-[var(--color-gold-soft)]">
                ≈ {duration((selected.xpToNext - selected.xpIntoLevel) / calcEffective)}
              </span>
            </div>
            <div class="tabular mt-1.5 flex items-center justify-between">
              <span class="text-[var(--color-muted)]">Time to {selected.next.label}</span>
              <span class="font-medium text-[var(--color-jade)]">
                ≈ {duration(selected.next.remaining / calcEffective)}
              </span>
            </div>
            <p class="mt-2 text-[10px] text-[var(--color-faint)]">
              At {num(calcEffective)} xp/hour
            </p>
          </div>
        {:else}
          <p class="mt-3 text-[11px] text-[var(--color-rose)]">
            No estimates — enter an XP/hour value above.
          </p>
        {/if}
      </div>
    </div>
  </Modal>
{/if}
