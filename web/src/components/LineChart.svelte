<script lang="ts">
  import { onDestroy } from 'svelte';
  import {
    Chart,
    LineController,
    LineElement,
    PointElement,
    LinearScale,
    CategoryScale,
    Tooltip,
    Filler,
    Legend,
  } from 'chart.js';

  Chart.register(
    LineController,
    LineElement,
    PointElement,
    LinearScale,
    CategoryScale,
    Tooltip,
    Filler,
    Legend,
  );

  export interface Series {
    label: string;
    color: string;
    points: number[];
  }

  let {
    labels,
    series,
    height = 280,
    yFormat = (n: number) => String(n),
    tooltipValue,
    beginAtZero = false,
  }: {
    labels: string[];
    series: Series[];
    height?: number;
    yFormat?: (n: number) => string;
    tooltipValue?: (n: number) => string;
    beginAtZero?: boolean;
  } = $props();

  let canvas = $state<HTMLCanvasElement | null>(null);
  let chart: Chart | null = null;

  function hexA(hex: string, a: number): string {
    const m = /^#?([0-9a-f]{6})$/i.exec(hex.trim());
    if (!m) return hex;
    const n = parseInt(m[1], 16);
    return `rgba(${(n >> 16) & 255}, ${(n >> 8) & 255}, ${n & 255}, ${a})`;
  }

  function datasets(list: Series[]) {
    return list.map((s) => ({
      label: s.label,
      data: s.points,
      borderColor: s.color,
      borderWidth: 2,
      tension: 0.32,
      fill: true,
      backgroundColor: (ctx: { chart: Chart }) => {
        const area = ctx.chart.chartArea;
        if (!area) return hexA(s.color, 0.12);
        const g = ctx.chart.ctx.createLinearGradient(0, area.top, 0, area.bottom);
        g.addColorStop(0, hexA(s.color, 0.28));
        g.addColorStop(1, hexA(s.color, 0));
        return g;
      },
    }));
  }

  $effect(() => {
    const l = labels;
    const s = series;
    const h = height;
    const yf = yFormat;
    const tv = tooltipValue;
    const bz = beginAtZero;
    if (!canvas) return;

    if (!chart) {
      chart = new Chart(canvas, {
        type: 'line',
        data: { labels: l, datasets: datasets(s) },
        options: {
          responsive: true,
          maintainAspectRatio: false,
          animation: { duration: 380 },
          interaction: { mode: 'index', intersect: false },
          plugins: {
            legend: {
              display: s.length > 1,
              position: 'top',
              align: 'end',
              labels: {
                color: '#8b98b0',
                boxWidth: 10,
                boxHeight: 10,
                usePointStyle: true,
                pointStyle: 'circle',
                font: { size: 11 },
              },
            },
            tooltip: {
              backgroundColor: '#0e131e',
              borderColor: '#2a3550',
              borderWidth: 1,
              padding: 10,
              titleColor: '#e8edf6',
              bodyColor: '#c6cfe0',
              usePointStyle: true,
              callbacks: {
                label: (item) =>
                  ` ${item.dataset.label}: ${(tv ?? yf)(Number(item.parsed.y))}`,
              },
            },
          },
          scales: {
            x: {
              grid: { color: 'rgba(255,255,255,0.04)' },
              border: { color: '#1f293c' },
              ticks: { color: '#8b98b0', maxRotation: 0, autoSkipPadding: 28, font: { size: 11 } },
            },
            y: {
              beginAtZero: bz,
              grid: { color: 'rgba(255,255,255,0.06)' },
              border: { display: false },
              ticks: {
                color: '#8b98b0',
                font: { size: 11 },
                callback: (v) => yf(Number(v)),
              },
            },
          },
          elements: { point: { radius: 0, hoverRadius: 4, hitRadius: 14 } },
        },
      });
    } else {
      chart.data.labels = l;
      chart.data.datasets = datasets(s);
      const yScale = chart.options.scales?.y;
      if (yScale) yScale.beginAtZero = bz;
      chart.update();
    }
  });

  onDestroy(() => {
    chart?.destroy();
    chart = null;
  });
</script>

<div class="relative w-full" style="height:{height}px">
  {#if series.length === 0 || series.every((s) => s.points.length === 0)}
    <div class="flex h-full items-center justify-center text-sm text-[var(--color-faint)]">
      No data for this range yet
    </div>
  {:else}
    <canvas bind:this={canvas}></canvas>
  {/if}
</div>
