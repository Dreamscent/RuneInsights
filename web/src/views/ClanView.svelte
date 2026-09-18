<script lang="ts">
  import { store } from '../lib/store.svelte';
  import { api, type ClanResponse } from '../lib/api';
  import Spinner from '../components/Spinner.svelte';
  import EmptyState from '../components/EmptyState.svelte';
  import { compact, num } from '../lib/format';

  const CLAN_RANKS = [
    'Owner',
    'Deputy owner',
    'General',
    'Captain',
    'Lieutenant',
    'Sergeant',
    'Corporal',
    'Recruit',
  ];

  let query = $state('');
  let data = $state<ClanResponse | null>(null);
  let loading = $state(false);
  let error = $state<string | null>(null);

  const trackedLower = $derived(new Set(store.players.map((p) => p.name.toLowerCase())));
  const rankLabel = (r: number) => CLAN_RANKS[Math.min(Math.max(r, 0), CLAN_RANKS.length - 1)];

  async function lookup(e: SubmitEvent) {
    e.preventDefault();
    const name = query.trim();
    if (!name || loading) return;
    loading = true;
    error = null;
    try {
      data = await api.clan(name);
    } catch (err) {
      error = err instanceof Error ? err.message : 'Lookup failed';
      data = null;
    } finally {
      loading = false;
    }
  }

  async function track(name: string) {
    await store.addPlayer(name, 'normal');
  }
</script>

<div class="flex max-w-2xl flex-col gap-5">
  <form class="flex gap-2" onsubmit={lookup}>
    <input
      class="input"
      placeholder="Clan name (exact)"
      bind:value={query}
      autocomplete="off"
      spellcheck="false"
    />
    <button type="submit" class="btn" disabled={loading || !query.trim()}>
      {loading ? 'Looking up…' : 'Look up'}
    </button>
  </form>

  {#if loading}
    <div class="flex justify-center py-12"><Spinner size={22} /></div>
  {:else if error}
    <div class="card">
      <EmptyState title="Clan not found" message={error} />
    </div>
  {:else if data}
    <div class="card overflow-hidden">
      <div class="flex items-center justify-between border-b border-[var(--color-line)] px-4 py-2.5">
        <h2 class="text-sm font-semibold text-[var(--color-ink)]">
          {data.clan}
          <span class="ml-2 text-xs font-normal text-[var(--color-muted)]">
            {data.members.length} members
          </span>
        </h2>
      </div>
      <div class="overflow-x-auto">
        <table class="w-full text-sm">
          <thead>
            <tr class="border-b border-[var(--color-line)] text-left text-[11px] tracking-wider text-[var(--color-faint)] uppercase">
              <th class="px-4 py-2 font-medium">Rank</th>
              <th class="px-3 py-2 font-medium">Member</th>
              <th class="px-3 py-2 text-right font-medium">Combat</th>
              <th class="px-3 py-2 text-right font-medium">Total XP</th>
              <th class="px-3 py-2 text-right font-medium"></th>
            </tr>
          </thead>
          <tbody class="divide-row">
            {#each data.members as m (m.name)}
              <tr class="transition-colors hover:bg-[var(--color-panel-2)]">
                <td class="px-4 py-2 text-[var(--color-muted)]">
                  <span class="chip !py-0 text-[10px]">{rankLabel(m.rank)}</span>
                </td>
                <td class="px-3 py-2 font-medium text-[var(--color-ink)]">{m.name}</td>
                <td class="tabular px-3 py-2 text-right text-[var(--color-muted)]">{m.kills ? num(m.kills) : '—'}</td>
                <td class="tabular px-3 py-2 text-right text-[var(--color-ink)]">{compact(m.overallXp)}</td>
                <td class="px-3 py-2 text-right">
                  {#if trackedLower.has(m.name.toLowerCase())}
                    <span class="text-[11px] text-[var(--color-jade)]">tracked</span>
                  {:else}
                    <button
                      class="btn-ghost text-[11px] text-[var(--color-gold)] hover:underline"
                      onclick={() => void track(m.name)}
                    >
                      + track
                    </button>
                  {/if}
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    </div>
    <p class="text-center text-[11px] text-[var(--color-faint)]">
      Members are listed by clan rank (Owner first), straight from the official clan hiscores.
    </p>
  {:else}
    <div class="card">
      <EmptyState
        title="Look up a clan"
        message="Enter any RuneScape 3 clan name to browse its member list with total XP, then track members with one click."
      />
    </div>
  {/if}
</div>
