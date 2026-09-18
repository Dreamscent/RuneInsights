<script lang="ts">
  import { store } from '../lib/store.svelte';
  import { api, type LeaderboardResponse } from '../lib/api';
  import Spinner from '../components/Spinner.svelte';
  import EmptyState from '../components/EmptyState.svelte';
  import { compact, num } from '../lib/format';
  import { skillColor } from '../lib/skills';

  // RS3 hiscore skill ids (matches hiscore index_lite ids)
  const SKILLS: { id: number; name: string }[] = [
    { id: 0, name: 'Overall' },
    { id: 1, name: 'Attack' },
    { id: 2, name: 'Defence' },
    { id: 3, name: 'Strength' },
    { id: 4, name: 'Hitpoints' },
    { id: 5, name: 'Ranged' },
    { id: 6, name: 'Prayer' },
    { id: 7, name: 'Magic' },
    { id: 8, name: 'Cooking' },
    { id: 9, name: 'Woodcutting' },
    { id: 10, name: 'Fletching' },
    { id: 11, name: 'Fishing' },
    { id: 12, name: 'Firemaking' },
    { id: 13, name: 'Crafting' },
    { id: 14, name: 'Smithing' },
    { id: 15, name: 'Mining' },
    { id: 16, name: 'Herblore' },
    { id: 17, name: 'Agility' },
    { id: 18, name: 'Thieving' },
    { id: 19, name: 'Slayer' },
    { id: 20, name: 'Farming' },
    { id: 21, name: 'Runecraft' },
    { id: 22, name: 'Hunter' },
    { id: 23, name: 'Construction' },
    { id: 24, name: 'Summoning' },
    { id: 25, name: 'Dungeoneering' },
    { id: 26, name: 'Divination' },
    { id: 27, name: 'Invention' },
    { id: 28, name: 'Archaeology' },
    { id: 29, name: 'Necromancy' },
  ];

  const CATEGORIES = [
    { id: 0, label: 'Skills' },
  ];

  let tableId = $state(0);
  let categoryId = $state(0);
  let data = $state<LeaderboardResponse | null>(null);
  let loading = $state(true);
  let error = $state<string | null>(null);

  async function load() {
    loading = true;
    error = null;
    try {
      data = await api.leaderboard(String(tableId), categoryId === 0 ? '0' : '1');
    } catch (e) {
      error = e instanceof Error ? e.message : 'Failed to load leaderboard';
    } finally {
      loading = false;
    }
  }

  $effect(() => {
    void tableId;
    void categoryId;
    void load();
  });

  const trackedLower = $derived(new Set(store.players.map((p) => p.name.toLowerCase())));
</script>

<div class="flex flex-col gap-4">
  <div class="flex flex-wrap items-center gap-2">
    <select class="input !w-auto" bind:value={tableId} aria-label="Skill">
      {#each SKILLS as s (s.id)}
        <option value={s.id}>{s.name}</option>
      {/each}
    </select>

    <div class="card inline-flex overflow-hidden p-1">
      {#each CATEGORIES as c (c.id)}
        <button
          class="rounded-lg px-4 py-1.5 text-xs font-medium capitalize transition-colors
            {categoryId === c.id
            ? 'bg-[var(--color-panel-2)] text-[var(--color-gold-soft)]'
            : 'text-[var(--color-muted)] hover:text-[var(--color-ink)]'}"
          onclick={() => (categoryId = c.id)}
        >
          {c.label}
        </button>
      {/each}
    </div>

    <span class="text-xs text-[var(--color-faint)]">Top 50 · RS3 global hiscores</span>
  </div>

  <div class="card overflow-hidden">
    {#if loading}
      <div class="flex justify-center py-12"><Spinner /></div>
    {:else if error}
      <EmptyState title="Leaderboard unavailable" message={error} />
    {:else if (data?.rows ?? []).length === 0}
      <EmptyState title="No results" message="Try another table." />
    {:else}
      <div class="overflow-x-auto">
        <table class="w-full text-sm">
          <thead>
            <tr class="border-b border-[var(--color-line)] text-left text-[11px] tracking-wider text-[var(--color-faint)] uppercase">
              <th class="px-4 py-2 font-medium">#</th>
              <th class="px-3 py-2 font-medium">Player</th>
              {#if categoryId === 0}
                <th class="px-3 py-2 text-right font-medium">Level</th>
              {/if}
              <th class="px-3 py-2 text-right font-medium">{categoryId === 0 ? 'Experience' : 'Score'}</th>
            </tr>
          </thead>
          <tbody class="divide-row">
            {#each data!.rows as row (row.rank)}
              <tr class="transition-colors hover:bg-[var(--color-panel-2)]">
                <td class="tabular px-4 py-2 text-[var(--color-faint)]">{row.rank}</td>
                <td class="px-3 py-2">
                  <span class="font-medium text-[var(--color-ink)]">{row.name}</span>
                  {#if trackedLower.has(row.name.toLowerCase())}
                    <span class="chip ml-2 !py-0 text-[10px] text-[var(--color-gold)]">tracked</span>
                  {/if}
                </td>
                {#if categoryId === 0}
                  <td class="tabular px-3 py-2 text-right text-[var(--color-muted)]">{row.level}</td>
                {/if}
                <td class="tabular px-3 py-2 text-right text-[var(--color-ink)]">
                  {compact(row.score)}
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    {/if}
  </div>
</div>
