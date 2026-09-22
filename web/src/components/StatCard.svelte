<script lang="ts">
  import type { Snippet } from 'svelte';

  let {
    label,
    value,
    valueSuffix,
    sub,
    accent = 'var(--color-gold)',
    icon,
    footnote,
  }: {
    label: string;
    value: string;
    /** small muted denominator after the main value (e.g. "/ 152") */
    valueSuffix?: string;
    sub?: string;
    accent?: string;
    icon?: Snippet;
    footnote?: string;
  } = $props();
</script>

<div class="card card-hover relative overflow-hidden p-4">
  <div
    class="pointer-events-none absolute -top-16 -right-12 h-32 w-32 rounded-full opacity-25 blur-2xl"
    style="background:{accent}"
  ></div>
  <div class="relative flex items-start justify-between gap-3">
    <div class="min-w-0">
      <div class="text-[11px] font-medium tracking-wide text-[var(--color-muted)] uppercase">
        {label}
      </div>
      <div class="tabular mt-1.5 text-2xl leading-tight font-semibold text-[var(--color-ink)]">
        {value}<span class="ml-1.5 text-base font-normal text-[var(--color-faint)]">{valueSuffix ?? ''}</span>
      </div>
      {#if sub}
        <div class="mt-1 truncate text-xs text-[var(--color-muted)]">{sub}</div>
      {/if}
    </div>
    {#if icon}
      <div class="shrink-0">{@render icon()}</div>
    {/if}
  </div>
  {#if footnote}
    <div class="relative mt-3 text-[11px] text-[var(--color-faint)]">{footnote}</div>
  {/if}
</div>
