<script setup lang="ts">
import { computed } from 'vue'
import type { StatsPoint } from '../types'

const props = defineProps<{ series: StatsPoint[] }>()

const WIDTH = 720
const HEIGHT = 240
const PAD = { top: 12, right: 8, bottom: 28, left: 36 }

const maxClicks = computed(() => Math.max(1, ...props.series.map((p) => p.clicks)))

const bars = computed(() => {
  const n = Math.max(props.series.length, 1)
  const innerW = WIDTH - PAD.left - PAD.right
  const innerH = HEIGHT - PAD.top - PAD.bottom
  const step = innerW / n
  const barW = Math.max(2, step * 0.6)
  return props.series.map((p, i) => {
    const h = (p.clicks / maxClicks.value) * innerH
    return {
      date: p.date,
      clicks: p.clicks,
      x: PAD.left + i * step + (step - barW) / 2,
      y: HEIGHT - PAD.bottom - h,
      w: barW,
      h,
    }
  })
})

const yTicks = computed(() => {
  const ticks: { value: number; y: number }[] = []
  const innerH = HEIGHT - PAD.top - PAD.bottom
  const steps = 4
  for (let i = 0; i <= steps; i++) {
    const value = Math.round((maxClicks.value * i) / steps)
    ticks.push({ value, y: HEIGHT - PAD.bottom - (value / maxClicks.value) * innerH })
  }
  return ticks
})

const ariaLabel = computed(() => {
  if (!props.series.length) return 'Bar chart of clicks per day. No clicks yet.'
  return `Bar chart of clicks per day, from ${props.series[0].date} to ${props.series[props.series.length - 1].date}, peak ${maxClicks.value} clicks.`
})

function shortDate(date: string): string {
  return date.slice(5)
}
</script>

<template>
  <div class="chart-wrap">
    <svg
      :viewBox="`0 0 ${WIDTH} ${HEIGHT}`"
      class="chart"
      role="img"
      :aria-label="ariaLabel"
      preserveAspectRatio="xMidYMid meet"
    >
      <g v-for="tick in yTicks" :key="tick.value">
        <line :x1="PAD.left" :x2="WIDTH - PAD.right" :y1="tick.y" :y2="tick.y" class="gridline" />
        <text :x="PAD.left - 6" :y="tick.y + 4" class="tick" text-anchor="end">{{ tick.value }}</text>
      </g>
      <rect
        v-for="bar in bars"
        :key="bar.date"
        :x="bar.x"
        :y="bar.y"
        :width="bar.w"
        :height="Math.max(bar.h, bar.clicks > 0 ? 2 : 1)"
        :class="['bar', { empty: bar.clicks === 0 }]"
      >
        <title>{{ bar.date }}: {{ bar.clicks }} clicks</title>
      </rect>
      <text
        v-for="(bar, i) in bars.filter((_, i2) => bars.length <= 10 || i2 % Math.ceil(bars.length / 8) === 0)"
        :key="'l' + i"
        :x="bar.x + bar.w / 2"
        :y="HEIGHT - 8"
        class="tick"
        text-anchor="middle"
      >{{ shortDate(bar.date) }}</text>
    </svg>
    <p v-if="series.every((p) => p.clicks === 0)" class="no-clicks">No clicks yet</p>
  </div>
</template>

<style scoped>
.chart-wrap {
  position: relative;
  width: 100%;
  height: 180px;
}
@media (min-width: 768px) {
  .chart-wrap {
    height: 220px;
  }
}
@media (min-width: 1280px) {
  .chart-wrap {
    height: 260px;
  }
}
.chart {
  width: 100%;
  height: 100%;
}
.gridline {
  stroke: var(--color-border);
  stroke-width: 1;
}
.tick {
  font-size: 11px;
  fill: var(--color-muted);
  font-family: var(--font-sans);
}
.bar {
  fill: var(--color-primary);
}
.bar.empty {
  fill: var(--color-surface-muted);
}
.no-clicks {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--color-muted);
  font-size: var(--text-sm);
  pointer-events: none;
}
</style>
