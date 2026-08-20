<template>
  <div class="flex gap-1">
    <button class="control-btn minimize" :title="t('common.minimize')" @click="emit('minimize')">
      <Minimize2 :size="12" />
    </button>
    <button class="control-btn maximize" :title="t('common.maximize')" @click="emit('maximize')">
      <Maximize2 :size="12" />
    </button>
    <button class="control-btn close" :title="t('common.close')" @click="emit('close')">
      <XIcon :size="16" />
    </button>
  </div>
</template>

<script setup lang="ts">
import { Maximize2, Minimize2, XIcon } from 'lucide-vue-next'

import { useTranslation } from '../../composables/useI18n'

const { t } = useTranslation()

const emit = defineEmits(['minimize', 'maximize', 'close'])
</script>

<style scoped>
@import 'tailwindcss' reference;
@import '../../assets/css/index.css' reference;

.control-btn {
  @apply flex h-5 w-7 cursor-pointer items-center justify-center rounded-sm border-0 bg-transparent text-secondary transition-all duration-fast ease-standard hover:bg-surface-elevated hover:text-main [.close:hover]:bg-danger [.close:hover]:text-white [.maximize:hover]:bg-success/85 [.maximize:hover]:text-white [.minimize:hover]:bg-accent/85 [.minimize:hover]:text-white;
}
</style>
