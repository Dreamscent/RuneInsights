<script lang="ts">
  import Modal from './Modal.svelte';
  import { store } from '../lib/store.svelte';
  import type { AccountType } from '../lib/api';

  let { open, onclose }: { open: boolean; onclose: () => void } = $props();

  let name = $state('');
  let accountType = $state<AccountType>('normal');
  let submitting = $state(false);

  $effect(() => {
    if (open) {
      name = '';
      accountType = 'normal';
      submitting = false;
    }
  });

  async function submit(e: SubmitEvent) {
    e.preventDefault();
    const trimmed = name.trim();
    if (!trimmed || submitting) return;
    submitting = true;
    const ok = await store.addPlayer(trimmed, accountType);
    submitting = false;
    if (ok) onclose();
  }
</script>

<Modal
  {open}
  {onclose}
  title="Track a player"
  subtitle="Look up any RuneScape 3 player by their display name."
>
  <form class="flex flex-col gap-4" onsubmit={submit}>
    <div>
      <label class="mb-1.5 block text-xs font-medium text-[var(--color-muted)]" for="add-name">
        Display name
      </label>
      <input
        id="add-name"
        class="input"
        placeholder="e.g. Zezima"
        bind:value={name}
        autocomplete="off"
        spellcheck="false"
        maxlength={12}
        required
      />
    </div>

    <div>
      <span class="mb-1.5 block text-xs font-medium text-[var(--color-muted)]">Account type</span>
      <div class="grid grid-cols-3 gap-2">
        {#each ['normal', 'ironman', 'hardcore'] as const as t (t)}
          <button
            type="button"
            class="rounded-lg border px-3 py-2 text-xs font-medium capitalize transition-colors
              {accountType === t
              ? 'border-[var(--color-gold)] bg-[rgba(245,165,36,.11)] text-[var(--color-gold-soft)]'
              : 'border-[var(--color-line-2)] text-[var(--color-muted)] hover:border-[#3b4a6b]'}"
            onclick={() => (accountType = t)}
          >
            {t}
          </button>
        {/each}
      </div>
      <p class="mt-1.5 text-[11px] text-[var(--color-faint)]">
        {accountType === 'normal'
          ? 'Standard account, uses the main hiscore table.'
          : accountType === 'ironman'
            ? 'Reads from the Ironman hiscore table.'
            : 'Reads from the Hardcore Ironman hiscore table.'}
      </p>
    </div>

    {#if store.mutating && open}
      <p class="text-xs text-[var(--color-muted)]">Looking up {name.trim()}…</p>
    {/if}

    <div class="flex items-center justify-end gap-2 pt-1">
      <button type="button" class="btn" onclick={onclose}>Cancel</button>
      <button type="submit" class="btn btn-gold" disabled={submitting || !name.trim()}>
        {submitting ? 'Looking up…' : 'Track player'}
      </button>
    </div>
  </form>
</Modal>
