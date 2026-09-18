<script lang="ts">
  import { store } from '../lib/store.svelte';

  const accent = (kind: string) =>
    kind === 'ok'
      ? 'var(--color-jade)'
      : kind === 'err'
        ? 'var(--color-rose)'
        : 'var(--color-sky)';
</script>

<div class="pointer-events-none fixed right-4 bottom-4 z-50 flex w-80 flex-col gap-2">
  {#each store.toasts as t (t.id)}
    <div
      class="card pointer-events-auto flex items-start gap-3 p-3 shadow-2xl animate-in"
      style="border-color:color-mix(in srgb, {accent(t.kind)} 45%, transparent)"
      role="alert"
    >
      <span
        class="mt-1 h-2.5 w-2.5 shrink-0 rounded-full"
        style="background:{accent(t.kind)}"
      ></span>
      <p class="flex-1 text-sm text-[var(--color-ink)]">{t.message}</p>
      <button
        class="btn-ghost rounded-md px-1.5 text-[var(--color-faint)] hover:text-[var(--color-ink)]"
        aria-label="Dismiss"
        onclick={() => store.dismiss(t.id)}
      >
        ✕
      </button>
    </div>
  {/each}
</div>
