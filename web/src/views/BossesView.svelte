<script lang="ts">
  import { store } from '../lib/store.svelte';
  import { api, type ActivitiesResponse, type Period, type Rates } from '../lib/api';
  import SkillIcon from '../components/SkillIcon.svelte';
  import Spinner from '../components/Spinner.svelte';
  import EmptyState from '../components/EmptyState.svelte';
  import { compact, num, signedCompact, timeAgo } from '../lib/format';
  import { skillColor } from '../lib/skills';

  const PERIODS: Period[] = ['day', 'week', 'month', 'year'];
  let period = $state<Period>('week');

  let data = $state<ActivitiesResponse | null>(null);
  let gains = $state<Map<string, number> | null>(null);
  let loading = $state(true);
  let onlyActive = $state(true);

  async function load(id: number) {
    loading = true;
    try {
      const resp = await api.activities(id, period);
      data = resp;
      gains = new Map(Object.entries(resp.gains ?? {}));
    } catch (e) {
      store.notify(e instanceof Error ? e.message : 'Failed to load activities', 'err');
    } finally {
      loading = false;
    }
  }

  $effect(() => {
    const id = store.selectedId;
    if (id == null) return;
    void period;
    void load(id);
  });

  const rows = $derived.by(() => {
    const all = (data?.activities ?? [])
      .filter((a) => (onlyActive ? (a.score ?? 0) > 0 : true))
      .sort((x, y) => (y.rank ?? Number.MAX_SAFE_INTEGER) - (x.rank ?? Number.MAX_SAFE_INTEGER));
    return all;
  });

  const sections = $derived.by(() => {
    const list = rows;
    const bosses: typeof list = [];
    const minigames: typeof list = [];
    for (const a of list) {
      if (/clue|lms|arena|soul wars|zeal|sc|caste|dominion|crucible|duel|fist of guthix|maze|bars|treasure|wilderness|penguin|cabbage|heist| disturbi/i.test(a.name))
        minigames.push(a);
      else bosses.push(a);
    }
    return { bosses, minigames };
  });
</script>

<div class="flex flex-col gap-5">
  {#if !store.skills}
    <div class="flex justify-center py-16"><Spinner /></div>
  {:else}
    <div class="flex items-center gap-2">
      <div class="card inline-flex overflow-hidden p-1">
        {#each PERIODS as p (p)}
          <button
            class="rounded-lg px-4 py-1.5 text-xs font-medium capitalize transition-colors
              {period === p
              ? 'bg-[var(--color-panel-2)] text-[var(--color-gold-soft)]'
              : 'text-[var(--color-muted)] hover:text-[var(--color-ink)]'}"
            onclick={() => (period = p)}
          >
            {p}
          </button>
        {/each}
      </div>
      <label class="chip cursor-pointer select-none gap-2">
        <input type="checkbox" class="accent-[var(--color-gold)]" bind:checked={onlyActive} />
        only show tracked
      </label>
    </div>

    {#if loading}
      <div class="flex justify-center py-12"><Spinner size={22} /></div>
    {:else if rows.length === 0}
      <div class="card">
        <EmptyState title="No activity data" message="No bosses or minigames have been captured yet." />
      </div>
    {:else}
      {#each ['bosses', 'minigames'] as section}
        {@const list = section === 'bosses' ? sections.bosses : sections.minigames}
        {#if list.length > 0}
          <div class="card overflow-hidden">
            <div class="flex items-center justify-between border-b border-[var(--color-line)] px-4 py-2.5">
              <h2 class="text-sm font-semibold capitalize text-[var(--color-ink)]">{section === 'bosses' ? 'Bosses & Raids' : 'Minigames & Other'}</h2>
              <span class="text-[11px] text-[var(--color-faint)]">{list.length} tracked</span>
            </div>
            <div class="overflow-x-auto">
              <table class="w-full text-sm">
                <thead>
                  <tr class="border-b border-[var(--color-line)] text-left text-[11px] tracking-wider text-[var(--color-faint)] uppercase">
                    <th class="px-4 py-2 font-medium">Name</th>
                    <th class="px-3 py-2 text-right font-medium">Score</th>
                    <th class="px-3 py-2 text-right font-medium">Rank</th>
                    <th class="px-3 py-2 text-right font-medium">Change ({period})</th>
                  </tr>
                </thead>
                <tbody class="divide-row">
                  {#each list as a (a.key)}
                    {@const color = skillColor(a.key)}
                    <tr class="transition-colors hover:bg-[var(--color-panel-2)]">
                      <td class="px-4 py-2">
                        <div class="flex items-center gap-2.5">
                          <span class="h-2.5 w-2.5 rounded-full" style="background:{color.accent}"></span>
                          <span class="text-[var(--color-ink)]">{a.name}</span>
                        </div>
                      </td>
                      <td class="tabular px-3 py-2 text-right text-[var(--color-ink)]">
                        {a.score ? num(a.score) : '—'}
                      </td>
                      <td class="tabular px-3 py-2 text-right text-[var(--color-muted)]">
                        {a.rank ? num(a.rank) : '—'}
                      </td>
                      <td class="tabular px-3 py-2 text-right">
                        {#if gains && gains.has(a.key) && (gains.get(a.key) ?? 0) > 0}
                          <span class="text-[var(--color-jade)]">+{num(gains.get(a.key))}</span>
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
        {/if}
      {/each}

      <p class="text-center text-[11px] text-[var(--color-faint)]">
        Scores and ranks are stored with every snapshot{data ? ` · last captured ${timeAgo(data.player.lastFetchedAt)}` : ''}.
        Period change columns activate once snapshot history spans a full {period}.
      </p>
    {/if}
  {/if}
</div>
