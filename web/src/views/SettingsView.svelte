<script lang="ts">
  import { store } from '../lib/store.svelte';
  import type { AccountType } from '../lib/api';
  import Modal from '../components/Modal.svelte';
  import { dateTime } from '../lib/format';

  const player = $derived(store.selected);
  const data = $derived(store.skills);

  const INTERVALS = [15, 30, 60, 120, 240];

  let confirmDelete = $state(false);
  let savingType = $state<AccountType | null>(null);
  let savingInterval = $state<number | null>(null);

  async function setType(t: AccountType) {
    if (!player || player.accountType === t) return;
    savingType = t;
    await store.updateSelected({ accountType: t });
    savingType = null;
  }

  async function setInterval(min: number) {
    if (!player || player.intervalMin === min) return;
    savingInterval = min;
    await store.updateSelected({ intervalMin: min });
    savingInterval = null;
  }

  async function doDelete() {
    const ok = await store.removeSelected();
    confirmDelete = false;
  }
</script>

<div class="flex max-w-2xl flex-col gap-5">
  {#if player}
    <section class="card p-4">
      <h2 class="mb-1 text-sm font-semibold text-[var(--color-ink)]">Tracking</h2>
      <p class="mb-4 text-xs text-[var(--color-muted)]">
        Snapshots of the official hiscores are collected automatically.
      </p>

      <div class="grid grid-cols-2 gap-4 text-sm">
        <div>
          <div class="text-[11px] tracking-wider text-[var(--color-faint)] uppercase">Player</div>
          <div class="mt-1 font-medium">{player.name}</div>
        </div>
        <div>
          <div class="text-[11px] tracking-wider text-[var(--color-faint)] uppercase">Snapshots</div>
          <div class="tabular mt-1 font-medium">{data?.snapshotCount ?? 0}</div>
        </div>
        <div>
          <div class="text-[11px] tracking-wider text-[var(--color-faint)] uppercase">Last capture</div>
          <div class="mt-1">{dateTime(data?.snapshotAt ?? player.lastFetchedAt)}</div>
        </div>
        <div>
          <div class="text-[11px] tracking-wider text-[var(--color-faint)] uppercase">Tracking since</div>
          <div class="mt-1">{dateTime(data?.collectingSince ?? player.createdAt)}</div>
        </div>
      </div>

      <div class="mt-5">
        <div class="mb-2 text-[11px] tracking-wider text-[var(--color-faint)] uppercase">
          Account type (hiscore table)
        </div>
        <div class="grid grid-cols-3 gap-2">
          {#each ['normal', 'ironman', 'hardcore'] as const as t (t)}
            <button
              class="rounded-lg border px-3 py-2 text-xs font-medium capitalize transition-colors
                {player.accountType === t
                ? 'border-[var(--color-gold)] bg-[rgba(245,165,36,.11)] text-[var(--color-gold-soft)]'
                : 'border-[var(--color-line-2)] text-[var(--color-muted)] hover:border-[#3b4a6b]'}"
              disabled={savingType !== null}
              onclick={() => void setType(t)}
            >
              {t}
              {#if savingType === t}<span class="ml-1 animate-pulse">…</span>{/if}
            </button>
          {/each}
        </div>
        <p class="mt-1.5 text-[11px] text-[var(--color-faint)]">
          Changing this reads the next snapshots from the matching hiscore table.
        </p>
      </div>

      <div class="mt-5">
        <div class="mb-2 text-[11px] tracking-wider text-[var(--color-faint)] uppercase">
          Capture interval
        </div>
        <div class="flex flex-wrap gap-2">
          {#each INTERVALS as min (min)}
            <button
              class="rounded-lg border px-3.5 py-2 text-xs font-medium transition-colors
                {player.intervalMin === min
                ? 'border-[var(--color-gold)] bg-[rgba(245,165,36,.11)] text-[var(--color-gold-soft)]'
                : 'border-[var(--color-line-2)] text-[var(--color-muted)] hover:border-[#3b4a6b]'}"
              disabled={savingInterval !== null}
              onclick={() => void setInterval(min)}
            >
              {min >= 60 ? `${min / 60}h` : `${min}m`}
            </button>
          {/each}
        </div>
        <p class="mt-1.5 text-[11px] text-[var(--color-faint)]">
          More frequent captures give smoother daily/weekly graphs. Manual refresh is capped at once
          every 5 minutes to stay polite to Jagex's API.
        </p>
      </div>
    </section>

    <section class="card p-4">
      <h2 class="text-sm font-semibold text-[var(--color-ink)]">About</h2>
      <div class="mt-2 flex flex-col gap-1.5 text-xs text-[var(--color-muted)]">
        <p>• RuneInsights polls Jagex's official RuneScape 3 hiscores on a schedule and stores a snapshot locally (SQLite).</p>
        <p>• XP gains for day / week / month / year are deltas between your own snapshot history, so graphs improve the longer the server runs.</p>
        <p>• Invention is an elite skill with its own XP curve (99 = 36.07M XP, virtual cap 150), handled exactly.</p>
        <p>• Combat level uses the current RS3 formula (max 152).</p>
      </div>
    </section>

    <section class="card border-[rgba(251,113,133,0.35)] p-4">
      <h2 class="text-sm font-semibold text-[var(--color-rose)]">Danger zone</h2>
      <p class="mt-1 text-xs text-[var(--color-muted)]">
        Remove {player.name} from tracking. This permanently deletes every stored snapshot for them.
      </p>
      <button class="btn mt-3 border-[rgba(251,113,133,0.4)]" onclick={() => (confirmDelete = true)}>
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="var(--color-rose)" stroke-width="2" stroke-linecap="round">
          <path d="M4 7h16M10 11v6M14 11v6M6 7l1 13h10l1-13M9 7V4h6v3" />
        </svg>
        Stop tracking
      </button>
    </section>
  {:else}
    <p class="text-sm text-[var(--color-muted)]">Select a player first.</p>
  {/if}
</div>

<Modal
  open={confirmDelete}
  onclose={() => (confirmDelete = false)}
  title={`Stop tracking ${player?.name ?? ''}?`}
  subtitle="All stored snapshots and goals for this player will be deleted."
>
  <div class="flex justify-end gap-2">
    <button class="btn" onclick={() => (confirmDelete = false)}>Cancel</button>
    <button class="btn border-[rgba(251,113,133,0.5)] text-[var(--color-rose)]" onclick={() => void doDelete()}>
      Delete permanently
    </button>
  </div>
</Modal>
