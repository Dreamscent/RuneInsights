<script lang="ts">
  import type { Snippet } from 'svelte';

  let {
    open = true,
    title,
    subtitle,
    onclose,
    children,
    footer,
    width = '30rem',
  }: {
    open?: boolean;
    title: string;
    subtitle?: string;
    onclose: () => void;
    children: Snippet;
    footer?: Snippet;
    width?: string;
  } = $props();

  function onKey(e: KeyboardEvent) {
    if (e.key === 'Escape' && open) onclose();
  }
</script>

<svelte:window onkeydown={onKey} />

{#if open}
  <div
    class="fixed inset-0 z-40 flex items-start justify-center overflow-y-auto bg-black/60 p-4 backdrop-blur-sm sm:items-center"
    role="presentation"
    onclick={(e) => {
      if (e.target === e.currentTarget) onclose();
    }}
  >
    <div
      class="card my-auto w-full p-5 shadow-2xl animate-in"
      style="max-width:{width}"
      role="dialog"
      aria-modal="true"
      aria-label={title}
    >
      <div class="flex items-start justify-between gap-4">
        <div>
          <h2 class="text-base font-semibold text-[var(--color-ink)]">{title}</h2>
          {#if subtitle}
            <p class="mt-1 text-xs text-[var(--color-muted)]">{subtitle}</p>
          {/if}
        </div>
        <button
          class="btn-ghost -mt-1 rounded-md px-2 py-1 text-[var(--color-faint)]"
          aria-label="Close"
          onclick={onclose}>✕</button
        >
      </div>

      <div class="mt-4">{@render children()}</div>

      {#if footer}
        <div class="mt-5 flex items-center justify-end gap-2">{@render footer()}</div>
      {/if}
    </div>
  </div>
{/if}
