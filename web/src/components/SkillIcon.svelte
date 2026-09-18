<script lang="ts">
  import { skillColor } from '../lib/skills';

  let {
    skillKey,
    name,
    size = 34,
    rounded = 8,
    dim = false,
  }: {
    skillKey: string;
    name?: string;
    size?: number;
    rounded?: number;
    dim?: boolean;
  } = $props();

  const color = $derived(skillColor(skillKey));
  const iconSrc = $derived(`/skills/${skillKey}.png`);
  const hasIcon = $derived(skillKey !== 'overall');

  let failed = $state(false);
</script>

{#if hasIcon && !failed}
  <span
    class="inline-flex shrink-0 items-center justify-center overflow-hidden"
    style="width:{size}px;height:{size}px;border-radius:{rounded}px;background:rgba(255,255,255,0.05)"
  >
    <img
      src={iconSrc}
      alt={name ?? skillKey}
      width={size}
      height={size}
      style={dim ? 'filter:saturate(.55) brightness(.85);' : ''}
      class="h-full w-full object-contain [image-rendering:-webkit-optimize-contrast]"
      draggable="false"
      onerror={() => (failed = true)}
    />
  </span>
{:else}
  <span
    class="inline-flex shrink-0 select-none items-center justify-center bg-gradient-to-br font-semibold text-white/95"
    style="width:{size}px;height:{size}px;border-radius:{rounded}px;box-shadow:inset 0 1px 0 rgba(255,255,255,.25), 0 4px 12px -8px rgba(0,0,0,.95);background:{color.gradient};font-size:{Math.round(size * 0.44)}px"
    title={name ?? skillKey}
  >
    Σ
  </span>
{/if}
