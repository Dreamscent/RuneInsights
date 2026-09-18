<script lang="ts">
  import type { Snippet } from 'svelte';

  let {
    title,
    message,
    icon,
    action,
  }: {
    title: string;
    message?: string;
    icon?: Snippet;
    action?: Snippet;
  } = $props();
</script>

<div class="flex flex-col items-center justify-center px-6 py-14 text-center">
  {#if icon}
    <div class="mb-4">{@render icon()}</div>
  {/if}
  <h3 class="text-base font-semibold text-[var(--color-ink)]">{title}</h3>
  {#if message}
    <p class="mt-1.5 max-w-md text-sm text-[var(--color-muted)]">{message}</p>
  {/if}
  {#if action}
    <div class="mt-5">{@render action()}</div>
  {/if}
</div>
