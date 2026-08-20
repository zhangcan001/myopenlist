<template>
  <div class="flex flex-col gap-4 w-full justify-center p-4">
    <div class="flex gap-2 justify-start items-center">
      <Eye class="text-accent" />
      <h4 class="font-semibold text-main">{{ t('dashboard.coreMonitor.title') }}</h4>
      <div
        class="flex items-center gap-2.5 text-xs font-medium py-0.5 px-1.5 rounded-2xl"
        :class="{
          'bg-success/70 ': isCoreRunning,
          'bg-danger/50  text-white': !isCoreRunning,
        }"
      >
        <span class="text-xs">{{
          isCoreRunning ? t('dashboard.coreMonitor.online') : t('dashboard.coreMonitor.offline')
        }}</span>
      </div>
    </div>

    <div class="flex flex-col gap-2">
      <div class="flex justify-between items-center p=1">
        <div v-if="isCoreRunning" class="flex gap-4 text-xs font-medium">
          <span class="flex items-center gap-1 py-2 px-3.5 rounded-2xl bg-accent/10 border border-border-secondary">
            <Globe :size="14" />
            Port: {{ openlistCoreStatus.port || 5244 }}
          </span>
          <span class="flex items-center gap-1 py-2 px-3.5 rounded-2xl bg-accent/10 border border-border-secondary">
            <Activity :size="14" />
            {{ t('dashboard.coreMonitor.responseTime') }}: {{ responseTime }}ms
          </span>
          <span
            class="flex items-center gap-1 py-2 px-3.5 rounded-2xl bg-accent/10 border border-border-secondary"
            :class="{
              'bg-success/10': avgResponseTime < 100,
              'bg-warning/10': avgResponseTime >= 100 && avgResponseTime < 500,
              'bg-error/10': avgResponseTime >= 500,
            }"
          >
            {{ avgResponseTime }}ms avg
          </span>
          <span
            class="flex items-center gap-1 py-2 px-3.5 rounded-2xl border border-border-secondary"
            :class="{
              'bg-success/10': successRate >= 99,
              'bg-warning/10': successRate >= 95 && successRate < 99,
              'bg-error/10': successRate < 95,
            }"
            >{{ successRate }}% uptime</span
          >
        </div>
      </div>

      <div
        ref="chartContainer"
        class="relative w-full min-h-25 bg-bg-secondary overflow-visible rounded-md border border-border"
      >
        <svg :width="chartWidth" :height="chartHeight" class="heartbeat-svg">
          <defs>
            <pattern id="grid" width="20" height="20" patternUnits="userSpaceOnUse">
              <path d="M 20 0 L 0 0 0 20" fill="none" :stroke="gridColor" stroke-width="1" opacity="0.5" />
            </pattern>
          </defs>
          <rect width="100%" height="100%" fill="url(#grid)" />

          <path :d="heartbeatPath" fill="none" :stroke="lineColor" stroke-width="2" class="drop-shadow" />

          <circle
            v-for="(point, index) in visibleDataPoints"
            :key="index"
            :cx="point.x"
            :cy="point.y"
            :r="point.isHealthy ? 3 : 4"
            :fill="point.isHealthy ? lineColor : '#ef4444'"
            class="cursor-pointer transition-[r] duration-200 ease-in-out hover:r-[5px]"
            @mouseover="showTooltip(point, $event)"
            @mouseleave="hideTooltip"
          />
        </svg>

        <div
          v-if="tooltip.show"
          class="absolute z-10 pointer-events-none"
          :style="{ left: tooltip.x + 'px', top: tooltip.y + 'px' }"
        >
          <div class="bg-black/80 text-white p-2 rounded-sm text-xs shadow-sm">
            <div class="font-medium mb-1">{{ tooltip.time }}</div>
            <div class="text-accent">{{ tooltip.value }}ms</div>
            <div
              :class="{
                'text-success': tooltip.status === 'healthy',
                'text-error': tooltip.status === 'unhealthy',
              }"
            >
              {{ tooltip.statusText }}
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { Activity, Eye, Globe } from 'lucide-vue-next'
import { storeToRefs } from 'pinia'
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'

import { useTranslation } from '../../composables/useI18n'
import { useAppStore } from '../../stores/app'

const { t } = useTranslation()
const appStore = useAppStore()
const { isCoreRunning, openlistCoreStatus } = storeToRefs(appStore)

const chartContainer = ref<HTMLElement>()
const chartWidth = ref(400)
const chartHeight = ref(120)
const dataPoints = ref<{ timestamp: number; responseTime: number; isHealthy: boolean }[]>([])
const responseTime = ref(0)
const startTime = ref(Date.now())
const monitoringInterval = ref<number>()

const tooltip = ref({
  show: false,
  x: 0,
  y: 0,
  time: '',
  value: '',
  status: '',
  statusText: '',
})

const avgResponseTime = computed(() => {
  if (dataPoints.value.length === 0) return 0
  const sum = dataPoints.value.reduce((acc, point) => acc + point.responseTime, 0)
  return Math.round(sum / dataPoints.value.length)
})

const successRate = computed(() => {
  if (dataPoints.value.length === 0) return 100
  const healthyPoints = dataPoints.value.filter(point => point.isHealthy).length
  return Math.round((healthyPoints / dataPoints.value.length) * 100)
})

const visibleDataPoints = computed(() => {
  const maxPoints = Math.floor(chartWidth.value / 10)
  const recent = dataPoints.value.slice(-maxPoints)

  return recent.map((point, index) => {
    const x = (index / (recent.length - 1 || 1)) * (chartWidth.value - 20) + 10
    const normalizedResponse = Math.min(point.responseTime / 1000, 1) // Normalize to 0-1 (0-1000ms)
    const y = chartHeight.value - 20 - normalizedResponse * (chartHeight.value - 40)

    return {
      x,
      y,
      isHealthy: point.isHealthy,
      responseTime: point.responseTime,
      timestamp: point.timestamp,
    }
  })
})

const heartbeatPath = computed(() => {
  if (visibleDataPoints.value.length < 2) return ''

  let path = `M ${visibleDataPoints.value[0].x} ${visibleDataPoints.value[0].y}`

  for (let i = 1; i < visibleDataPoints.value.length; i++) {
    const prev = visibleDataPoints.value[i - 1]
    const curr = visibleDataPoints.value[i]
    const cpx1 = prev.x + (curr.x - prev.x) * 0.5
    const cpx2 = curr.x - (curr.x - prev.x) * 0.5
    path += ` C ${cpx1} ${prev.y} ${cpx2} ${curr.y} ${curr.x} ${curr.y}`
  }

  return path
})

const lineColor = computed(() => {
  if (avgResponseTime.value === 0) return '#6b7280'
  if (0 < avgResponseTime.value && avgResponseTime.value < 100) return '#10b981'
  if (avgResponseTime.value < 500) return '#f59e0b'
  return '#ef4444'
})

const gridColor = computed(() => {
  return document.documentElement.classList.contains('dark') ? '#64748b' : '#9ca3af'
})

const checkCoreHealth = async () => {
  await appStore.refreshOpenListCoreStatus()
  if (!isCoreRunning.value) {
    dataPoints.value.push({
      timestamp: Date.now(),
      responseTime: 0,
      isHealthy: false,
    })
    responseTime.value = 0
    return
  }

  const startTime = Date.now()

  try {
    await appStore.refreshOpenListCoreStatus()

    const endTime = Date.now()
    const responseTimeMs = endTime - startTime
    const isHealthy = responseTimeMs < 1000

    dataPoints.value.push({
      timestamp: endTime,
      responseTime: responseTimeMs,
      isHealthy,
    })

    responseTime.value = responseTimeMs

    if (dataPoints.value.length > 100) {
      dataPoints.value = dataPoints.value.slice(-100)
    }
  } catch (_error) {
    dataPoints.value.push({
      timestamp: Date.now(),
      responseTime: 5000,
      isHealthy: false,
    })
  }
}

const showTooltip = (point: any, event: MouseEvent) => {
  tooltip.value = {
    show: true,
    x: event.offsetX + 10,
    y: event.offsetY - 10,
    time: new Date(point.timestamp).toLocaleTimeString(),
    value: point.responseTime.toString(),
    status: point.isHealthy ? 'healthy' : 'unhealthy',
    statusText: point.isHealthy ? t('dashboard.coreMonitor.healthy') : t('dashboard.coreMonitor.unhealthy'),
  }
}

const hideTooltip = () => {
  tooltip.value.show = false
}

const updateChartSize = () => {
  if (chartContainer.value) {
    chartWidth.value = chartContainer.value.clientWidth
    chartHeight.value = chartContainer.value.clientHeight
  }
}

onMounted(async () => {
  await nextTick()
  updateChartSize()
  await appStore.refreshOpenListCoreStatus()

  if (isCoreRunning.value) {
    startTime.value = Date.now()
  }

  monitoringInterval.value = window.setInterval(checkCoreHealth, 15 * 1000)
  window.addEventListener('resize', updateChartSize)
})

onUnmounted(() => {
  if (monitoringInterval.value) {
    clearInterval(monitoringInterval.value)
  }
  window.removeEventListener('resize', updateChartSize)
})

watch(isCoreRunning, (newValue: boolean, oldValue: boolean) => {
  if (newValue && !oldValue) {
    startTime.value = Date.now()
  }
})
</script>
