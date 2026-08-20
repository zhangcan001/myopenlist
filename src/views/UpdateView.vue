<template>
  <div class="relative flex h-full w-full items-center justify-center">
    <div class="relative z-1 flex h-full w-full flex-col items-center justify-start gap-4 rounded-xl border-none p-4">
      <div
        class="flex w-full items-center justify-between gap-4 overflow-visible rounded-2xl border border-border-secondary px-4 py-2 shadow-md"
      >
        <div class="flex flex-1 flex-wrap items-center gap-4 p-1">
          <Settings :size="24" class="text-accent" />
          <div>
            <h1 class="m-0 text-xl font-semibold tracking-tight text-main">{{ t('update.title') }}</h1>
            <p class="m-0 text-xs text-secondary">{{ t('update.subtitle') }}</p>
          </div>
        </div>
      </div>

      <div
        class="relative flex h-full w-full flex-1 items-center justify-center overflow-hidden rounded-2xl border border-border-secondary p-4 shadow-md"
      >
        <UpdateManagerCard />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { Settings } from 'lucide-vue-next'

import UpdateManagerCard from '../components/dashboard/UpdateManagerCard.vue'
import { useTranslation } from '../composables/useI18n'

const { t } = useTranslation()
</script>
