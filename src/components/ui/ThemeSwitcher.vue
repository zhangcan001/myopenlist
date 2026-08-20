<template>
  <div class="relative flex items-center">
    <button
      class="flex cursor-pointer items-center gap-2 rounded-md border border-border-secondary bg-bg-secondary px-2 py-1.5 text-sm text-secondary transition-all duration-fast ease-standard hover:bg-accent/30 hover:text-white"
      :title="t('settings.theme.toggle')"
      @click="toggleTheme"
    >
      <component :is="currentThemeOption.icon" :size="18" />
      <span class="font-medium">{{ currentThemeOption.label }}</span>
    </button>
  </div>
</template>

<script setup lang="ts">
import { Monitor, Moon, Sun } from 'lucide-vue-next'
import { computed } from 'vue'

import { useTranslation } from '../../composables/useI18n'
import { useAppStore } from '../../stores/app'

const appStore = useAppStore()
const { t } = useTranslation()

const currentTheme = computed(() => appStore.settings.app.theme || 'light')

const themeOptions = computed(() => [
  {
    value: 'light',
    label: t('settings.theme.light'),
    icon: Sun,
    description: t('settings.theme.lightDesc'),
  },
  {
    value: 'dark',
    label: t('settings.theme.dark'),
    icon: Moon,
    description: t('settings.theme.darkDesc'),
  },
  {
    value: 'auto',
    label: t('settings.theme.auto'),
    icon: Monitor,
    description: t('settings.theme.autoDesc'),
  },
])

const currentThemeOption = computed(
  () => themeOptions.value.find(option => option.value === currentTheme.value) || themeOptions.value[0],
)

const toggleTheme = () => {
  appStore.toggleTheme()
}
</script>
