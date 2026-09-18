<script lang="ts">
  import { store, type ViewKey } from '../lib/store.svelte';

  interface NavItem {
    key: ViewKey;
    label: string;
    d: string;
    group: 'track' | 'explore' | 'config';
  }

  const NAV: NavItem[] = [
    { key: 'overview', label: 'Overview', group: 'track', d: 'M3 3h7v7H3zM14 3h7v5h-7zM14 12h7v9h-7zM3 14h7v7H3z' },
    { key: 'skills', label: 'Skills', group: 'track', d: 'M4 20V11M10 20V4M16 20v-6M22 20H2' },
    { key: 'progress', label: 'Progress', group: 'track', d: 'M3 17l5-5 4 3 8-8M21 7v5h-5' },
    { key: 'bosses', label: 'Bosses & Minigames', group: 'explore', d: 'M12 3l8 4v5c0 4.5-3.2 8.4-8 9.6C7.2 20.4 4 16.5 4 12V7zM12 8v5M9.5 10.5h5' },
    { key: 'leaderboard', label: 'Leaderboard', group: 'explore', d: 'M8 4h8v4a4 4 0 0 1-8 0zM8 5H5v2a3 3 0 0 0 3 3M16 5h3v2a3 3 0 0 1-3 3M12 12v4M9 20h6M10 16h4' },
    { key: 'clan', label: 'Clan', group: 'explore', d: 'M9 11a3 3 0 1 0 0-6 3 3 0 0 0 0 6zM2 20c0-3.3 3.1-5 7-5s7 1.7 7 5M17 11a3 3 0 1 0 0-6M18 15c2.5.4 4 1.6 4 4' },
    { key: 'rates', label: 'Training rates', group: 'config', d: 'M11 4H4v4h7zM19 4h-4v4h4zM11 10H4v4h11zM19 10h-4v4h4M11 16H4v4h7M19 20h-4M19 14v4M21 16l-2 2-1-1' },
    { key: 'settings', label: 'Settings', group: 'config', d: 'M12 15a3 3 0 1 0 0-6 3 3 0 0 0 0 6zM19.4 15a1.7 1.7 0 0 0 .3 1.9l.1.1a2 2 0 1 1-2.8 2.8l-.1-.1a1.7 1.7 0 0 0-1.9-.3 1.7 1.7 0 0 0-1 1.5V21a2 2 0 1 1-4 0v-.1a1.7 1.7 0 0 0-1.1-1.5 1.7 1.7 0 0 0-1.9.3l-.1.1a2 2 0 1 1-2.8-2.8l.1-.1a1.7 1.7 0 0 0 .3-1.9 1.7 1.7 0 0 0-1.5-1H3a2 2 0 1 1 0-4h.1a1.7 1.7 0 0 0 1.5-1.1 1.7 1.7 0 0 0-.3-1.9l-.1-.1a2 2 0 1 1 2.8-2.8l.1.1a1.7 1.7 0 0 0 1.9.3H9a1.7 1.7 0 0 0 1-1.5V3a2 2 0 1 1 4 0v.1a1.7 1.7 0 0 0 1 1.5 1.7 1.7 0 0 0 1.9-.3l.1-.1a2 2 0 1 1 2.8 2.8l-.1.1a1.7 1.7 0 0 0-.3 1.9V9a1.7 1.7 0 0 0 1.5 1H21a2 2 0 1 1 0 4h-.1a1.7 1.7 0 0 0-1.5 1z' },
  ];

  const groups: { id: NavItem['group']; label: string }[] = [
    { id: 'track', label: 'Track' },
    { id: 'explore', label: 'Explore' },
    { id: 'config', label: 'Configure' },
  ];

  let { onAddPlayer, onNavigated }: { onAddPlayer: () => void; onNavigated?: () => void } = $props();

  function go(key: ViewKey) {
    store.view = key;
    onNavigated?.();
  }
</script>

<aside
  class="flex h-full w-full flex-col gap-1 border-r border-[var(--color-line)] bg-[var(--color-bg-soft)]/85 px-3 py-4 backdrop-blur"
>
  <div class="mb-4 flex items-center gap-2.5 px-2">
    <span
      class="glow-gold flex h-9 w-9 items-center justify-center rounded-xl bg-gradient-to-br from-[#fbbf24] to-[#92400e] text-lg font-bold text-[#241703]"
      >R</span
    >
    <div class="leading-tight">
      <div class="text-sm font-semibold tracking-tight text-[var(--color-ink)]">RuneInsights</div>
      <div class="text-[10px] tracking-widest text-[var(--color-faint)] uppercase">
        RuneScape 3
      </div>
    </div>
  </div>

  {#each groups as g (g.id)}
    <div class="mt-2 px-2 text-[10px] font-semibold tracking-widest text-[var(--color-faint)] uppercase">
      {g.label}
    </div>
    <nav class="flex flex-col gap-0.5">
      {#each NAV.filter((n) => n.group === g.id) as item (item.key)}
        <button
          class="group flex items-center gap-2.5 rounded-lg px-2.5 py-2 text-left text-sm transition-colors
            {store.view === item.key
            ? 'bg-[var(--color-panel-2)] font-medium text-[var(--color-ink)] shadow-[inset_0_0_0_1px_var(--color-line-2)]'
            : 'text-[var(--color-muted)] hover:bg-[var(--color-panel)] hover:text-[var(--color-ink)]'}"
          onclick={() => go(item.key)}
        >
          <svg
            width="17"
            height="17"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="1.8"
            stroke-linecap="round"
            stroke-linejoin="round"
            class="shrink-0 {store.view === item.key ? 'text-[var(--color-gold)]' : ''}"
          >
            <path d={item.d} />
          </svg>
          <span class="truncate">{item.label}</span>
        </button>
      {/each}
    </nav>
  {/each}

  <div class="mt-auto pt-4">
    <button class="btn w-full justify-center" onclick={onAddPlayer}>
      <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
        <path d="M12 5v14M5 12h14" />
      </svg>
      Track player
    </button>
    <p class="mt-3 px-1 text-[10px] leading-relaxed text-[var(--color-faint)]">
      Data from the official RS3 hiscores. Snapshots are collected automatically.
    </p>
  </div>
</aside>
