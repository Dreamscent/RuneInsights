<script lang="ts">
  let {
    pctValue,
    color = 'var(--color-gold)',
    height = 8,
    track = 'rgba(255,255,255,0.06)',
    rounded = 999,
  }: {
    pctValue: number;
    color?: string;
    height?: number;
    track?: string;
    rounded?: number;
  } = $props();

  const clamped = $derived(Math.max(0, Math.min(100, Number.isFinite(pctValue) ? pctValue : 0)));
</script>

<div
  class="relative w-full overflow-hidden"
  style="height:{height}px;background:{track};border-radius:{rounded}px"
>
  <div
    class="h-full transition-[width] duration-500 ease-out"
    style="width:{clamped}%;background:{color};border-radius:{rounded}px"
  ></div>
</div>
