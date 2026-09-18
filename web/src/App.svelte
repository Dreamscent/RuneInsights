<script lang="ts">
  import type { Component } from 'svelte';
  import { store } from './lib/store.svelte';
  import Sidebar from './components/Sidebar.svelte';
  import TopBar from './components/TopBar.svelte';
  import Toasts from './components/Toasts.svelte';
  import AddPlayerDialog from './components/AddPlayerDialog.svelte';
  import Spinner from './components/Spinner.svelte';
  import EmptyState from './components/EmptyState.svelte';
  import Overview from './views/Overview.svelte';
  import SkillsView from './views/SkillsView.svelte';
    import ProgressView from './views/ProgressView.svelte';
  import BossesView from './views/BossesView.svelte';
  import LeaderboardView from './views/LeaderboardView.svelte';
  import ClanView from './views/ClanView.svelte';
  import SettingsView from './views/SettingsView.svelte';
  import TrainingRates from './views/TrainingRates.svelte';

  // createjs-style mapping of view -> component
  const VIEWS: Record<string, Component> = {
    overview: Overview,
    skills: SkillsView,
    progress: ProgressView,
    bosses: BossesView,
    leaderboard: LeaderboardView,
    clan: ClanView,
    settings: SettingsView,
    rates: TrainingRates,
  };

  let showAdd = $state(false);
  let mobileNav = $state(false);

  const view = $derived(VIEWS[store.view] ?? Overview);
  const ViewComp = $derived(view);
  const currentTitle = $derived.by(() => {
    switch (store.view) {
      case 'overview': return 'Overview';
      case 'skills': return 'Skills';
      case 'progress': return 'Progress';
      case 'bosses': return 'Bosses & Minigames';
      case 'leaderboard': return 'Leaderboard';
      case 'clan': return 'Clan';
      case 'settings': return 'Settings';
      case 'rates': return 'Training rates';
      default: return 'Overview';
    }
  });

  $effect(() => {
    void store.boot();
  });
</script>

<div class="flex h-screen overflow-hidden">
  <!-- desktop sidebar -->
  <div class="hidden w-60 shrink-0 lg:block">
    <Sidebar onAddPlayer={() => (showAdd = true)} onNavigated={() => (mobileNav = false)} />
  </div>

  <!-- mobile drawer -->
  {#if mobileNav}
    <div class="fixed inset-0 z-40 lg:hidden" role="presentation" onclick={() => (mobileNav = false)}>
      <div class="absolute inset-0 bg-black/60 backdrop-blur-sm"></div>
      <div
        class="absolute top-0 bottom-0 left-0 w-64 shadow-2xl"
        role="dialog"
        aria-modal="true"
        tabindex="-1"
        onkeydown={(e) => {
          if (e.key === 'Escape') mobileNav = false;
        }}
        onclick={(e) => e.stopPropagation()}
      >
        <Sidebar onAddPlayer={() => { showAdd = true; mobileNav = false; }} onNavigated={() => (mobileNav = false)} />
      </div>
    </div>
  {/if}

  <div class="flex min-w-0 flex-1 flex-col">
    <TopBar onAddPlayer={() => (showAdd = true)} />

    <div class="flex items-center gap-2 border-b border-[var(--color-line)] px-4 py-2 lg:hidden">
      <button class="btn-ghost" aria-label="Open navigation" onclick={() => (mobileNav = true)}>
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
          <path d="M4 6h16M4 12h16M4 18h16" />
        </svg>
      </button>
      <span class="text-sm font-semibold text-[var(--color-ink)]">{currentTitle}</span>
    </div>

    <main class="flex-1 overflow-y-auto">
      {#if store.booting}
        <div class="flex h-full items-center justify-center">
          <Spinner size={26} label="Loading your dashboard" />
        </div>
      {:else if !store.selected}
        <EmptyState
          title={store.players.length > 0 ? 'Select a player' : 'No players tracked yet'}
          message={store.players.length > 0
            ? 'Choose a player from the top bar, or track a new one.'
            : 'Add a RuneScape 3 display name and RuneInsights will snapshot their hiscores automatically — levels, XP, gains per day, and goal progress.'}
        >
          {#snippet icon()}
            <div
              class="glow-gold flex h-16 w-16 items-center justify-center rounded-2xl bg-gradient-to-br from-[#fbbf24] to-[#92400e] text-3xl font-bold text-[#241703]"
            >
              R
            </div>
          {/snippet}
          {#snippet action()}
            <button class="btn btn-gold px-5 py-2.5 text-sm" onclick={() => (showAdd = true)}>
              Track your first player
            </button>
          {/snippet}
        </EmptyState>
      {:else if store.skillsLoading && !store.skills}
        <div class="flex h-full items-center justify-center">
          <Spinner size={24} label={`Loading ${store.selected.name}`} />
        </div>
      {:else if store.globalError}
        <EmptyState
          title="Cannot reach the API"
          message={store.globalError}
        >
          {#snippet action()}
            <button class="btn" onclick={() => void store.boot()}>Try again</button>
          {/snippet}
        </EmptyState>
      {:else}
        <div class="mx-auto max-w-6xl px-4 py-6 sm:px-6">
          <h1 class="mb-5 hidden text-xl font-semibold tracking-tight text-[var(--color-ink)] lg:block">
            {currentTitle}
          </h1>
          {#key store.selectedId}
            <div class="animate-in">
              <ViewComp />
            </div>
          {/key}
        </div>
      {/if}
    </main>
  </div>
</div>

<AddPlayerDialog open={showAdd} onclose={() => (showAdd = false)} />
<Toasts />
