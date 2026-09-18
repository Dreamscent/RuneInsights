<script lang="ts">
  let { size = 18, label = 'Loading' }: { size?: number; label?: string } = $props();
</script>

<span class="inline-flex items-center gap-2 text-[var(--color-muted)]" role="status" aria-label={label}>
  <svg
    class="animate-spin"
    width={size}
    height={size}
    viewBox="0 0 24 24"
    fill="none"
    aria-hidden="true"
  >
    <circle cx="12" cy="12" r="9" stroke="currentColor" stroke-opacity="0.2" stroke-width="3" />
    <path
      d="M21 12a9 9 0 0 0-9-9"
      stroke="currentColor"
      stroke-width="3"
      stroke-linecap="round"
    />
  </svg>
  {#if label}<span class="text-sm">{label}</span>{/if}
</span>
