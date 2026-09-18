<script lang="ts">
  import { store } from '../lib/store.svelte';
  import { timeAgo } from '../lib/format';
  import SkillIcon from './SkillIcon.svelte';

  let { onAddPlayer }: { onAddPlayer: () => void } = $props();

  let menuOpen = $state(false);
  let root: HTMLElement | undefined = $state();

  // close on outside click
  $effect(() => {
    if (!menuOpen) return;
    const close = (e: Event) => {
      if (root && !root.contains(e.target as Node)) menuOpen = false;
    };
    document.addEventListener('mousedown', close);
    document.addEventListener('touchstart', close);
    return () => {
      document.removeEventListener('mousedown', close);
      document.removeEventListener('touchstart', close);
    };
  });

  const accountLabel: Record<string, string> = {
    normal: 'Standard',
    ironman: 'Ironman',
    hardcore: 'Hardcore',
  };
</script>

<header class="flex flex-wrap items-center gap-3 border-b border-[var(--color-line)] px-4 py-3 sm:px-6">
  <!-- player picker -->
  <div class="relative" bind:this={root}>
    {#if store.selected}
      <button
        class="card flex items-center gap-3 px-3 py-2 text-left card-hover"
        aria-haspopup="listbox"
        aria-expanded={menuOpen}
        onclick={() => (menuOpen = !menuOpen)}
      >
        <SkillIcon skillKey="overall" name={store.selected.name} size={30} rounded={8} />
        <span class="leading-tight">
          <span class="block max-w-40 truncate text-sm font-semibold text-[var(--color-ink)]">
            {store.selected.name}
          </span>
          <span class="text-[11px] text-[var(--color-muted)]">
            {accountLabel[store.selected.accountType] ?? 'Standard'} · updated
            {timeAgo(store.selected.lastFetchedAt)}
          </span>
        </span>
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="text-[var(--color-faint)]">
          <path d="M6 9l6 6 6-6" />
        </svg>
      </button>
    {:else}
      <button class="btn" onclick={onAddPlayer}>Track a player</button>
    {/if}

    {#if menuOpen}
      <div
        class="card absolute top-full left-0 z-30 mt-2 w-64 overflow-hidden p-1 shadow-2xl animate-in"
        role="listbox"
      >
        <div class="max-h-72 overflow-y-auto">
          {#if store.players.length === 0}
            <p class="px-3 py-2 text-sm text-[var(--color-muted)]">No players tracked yet.</p>
          {/if}
          {#each store.players as p (p.id)}
            <button
              class="flex w-full items-center justify-between gap-2 rounded-lg px-3 py-2 text-left text-sm
                {p.id === store.selectedId ? 'bg-[var(--color-panel-2)] text-[var(--color-ink)]' : 'text-[var(--color-muted)] hover:bg-[var(--color-panel)] hover:text-[var(--color-ink)]'}"
              onclick={() => {
                menuOpen = false;
                void store.select(p.id);
              }}
            >
              <span class="truncate font-medium">{p.name}</span>
              {#if p.id === store.selectedId}
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="var(--color-jade)" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
                  <path d="M5 13l4 4 10-11" />
                </svg>
              {/if}
            </button>
          {/each}
        </div>
        <div class="border-t border-[var(--color-line)] p-1">
          <button
            class="flex w-full items-center gap-2 rounded-lg px-3 py-2 text-left text-sm text-[var(--color-gold)] hover:bg-[var(--color-panel)]"
            onclick={() => {
              menuOpen = false;
              onAddPlayer();
            }}
          >
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
              <path d="M12 5v14M5 12h14" />
            </svg>
            Track another player
          </button>
        </div>
      </div>
    {/if}
  </div>

  <div class="flex-1"></div>

  {#if store.selected && store.skills}
    <span class="chip hidden sm:inline-flex">
      {store.skills.snapshotCount} snapshot{store.skills.snapshotCount === 1 ? '' : 's'}
    </span>
  {/if}

  <button
    class="btn"
    disabled={store.mutating || !store.selected}
    title="Fetch the latest hiscores now (max once every 5 minutes)"
    onclick={() => void store.refreshSelected()}
  >
    <svg
      width="15"
      height="15"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      stroke-width="2"
      stroke-linecap="round"
      stroke-linejoin="round"
      class={store.mutating ? 'animate-spin' : ''}
    >
      <path d="M21 12a9 9 0 1 1-2.64-6.36" />
      <path d="M21 3v6h-6" />
    </svg>
    Refresh
  </button>
</header>
