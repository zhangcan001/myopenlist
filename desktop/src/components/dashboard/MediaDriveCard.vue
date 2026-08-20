<template>
  <div class="flex w-full flex-col gap-4 p-4">
    <div class="flex flex-wrap items-center justify-between gap-2">
      <div class="flex items-center gap-2">
        <HardDrive class="text-accent" />
        <h4 class="font-semibold text-main">{{ t('dashboard.mediaDrive.title') }}</h4>
        <span class="rounded-2xl px-2 py-0.5 text-xs font-medium" :class="statusClass">
          {{ statusLabel }}
        </span>
      </div>
      <span class="text-xs text-secondary">{{ t('dashboard.mediaDrive.driveLetter') }}: {{ driveLetter }}</span>
    </div>

    <div class="grid grid-cols-3 gap-2 max-md:grid-cols-1">
      <div class="rounded-md border border-border-secondary bg-surface p-3">
        <div class="text-xs text-secondary">115</div>
        <div class="mt-1 flex items-center gap-1 text-sm font-medium text-main">
          <CircleCheck v-if="authReady" :size="15" class="text-success" />
          <CircleAlert v-else :size="15" class="text-warning" />
          {{ authReady ? t('dashboard.mediaDrive.connected') : t('dashboard.mediaDrive.notConnected') }}
        </div>
      </div>
      <div class="rounded-md border border-border-secondary bg-surface p-3">
        <div class="text-xs text-secondary">WebDAV</div>
        <div class="mt-1 flex items-center gap-1 text-sm font-medium text-main">
          <CircleCheck v-if="webdavReady" :size="15" class="text-success" />
          <CircleAlert v-else :size="15" class="text-warning" />
          {{ webdavReady ? t('dashboard.mediaDrive.running') : t('dashboard.mediaDrive.notRunning') }}
        </div>
      </div>
      <div class="rounded-md border border-border-secondary bg-surface p-3">
        <div class="text-xs text-secondary">{{ driveLetter }}</div>
        <div class="mt-1 flex items-center gap-1 text-sm font-medium text-main">
          <CircleCheck v-if="mountReady" :size="15" class="text-success" />
          <CircleAlert v-else :size="15" class="text-warning" />
          {{ mountReady ? t('dashboard.mediaDrive.mounted') : t('dashboard.mediaDrive.notMounted') }}
        </div>
      </div>
    </div>

    <div class="flex flex-wrap items-center gap-2 rounded-md border border-border-secondary bg-surface p-3">
      <span class="text-xs" :class="environment.winfsp_installed ? 'text-success' : 'text-warning'">
        {{ t('dashboard.mediaDrive.winfsp') }}:
        {{
          environment.winfsp_installed
            ? t('dashboard.mediaDrive.environmentReady')
            : t('dashboard.mediaDrive.environmentMissing')
        }}
      </span>
      <span v-if="isBusy" class="text-xs text-secondary">{{ statusLabel }}</span>
      <span v-if="stopped" class="text-xs text-secondary">{{ t('dashboard.mediaDrive.stopped') }}</span>
    </div>

    <div v-if="!config.configured" class="rounded-md border border-accent/40 bg-accent/5 p-4">
      <div class="mb-3 flex items-center gap-2">
        <Settings2 :size="17" class="text-accent" />
        <h5 class="font-semibold text-main">{{ t('dashboard.mediaDrive.setup') }}</h5>
      </div>
      <p class="mb-3 text-xs text-secondary">{{ t('dashboard.mediaDrive.setupHint') }}</p>
      <div class="grid gap-3 md:grid-cols-2">
        <label class="flex flex-col gap-1 text-xs text-secondary">
          {{ t('dashboard.mediaDrive.driveLetter') }}
          <input
            v-model="driveInput"
            maxlength="2"
            class="rounded-md border border-border bg-bg-secondary px-3 py-2 text-sm text-main"
          />
        </label>
        <label class="flex flex-col gap-1 text-xs text-secondary">
          {{ t('dashboard.mediaDrive.webdavPassword') }}
          <input
            v-model="webdavPassword"
            type="password"
            autocomplete="off"
            class="rounded-md border border-border bg-bg-secondary px-3 py-2 text-sm text-main"
          />
        </label>
      </div>
      <label class="mt-3 flex items-center gap-2 text-xs text-secondary">
        <input v-model="autoStartInput" type="checkbox" />
        {{ t('dashboard.mediaDrive.autoStart') }}
      </label>
      <CustomButton
        class="mt-3"
        type="primary"
        :icon="Play"
        :text="t('dashboard.mediaDrive.saveSetup')"
        :disabled="isBusy"
        @click="saveSetupAndStart"
      />
    </div>

    <div v-else class="flex flex-wrap items-end gap-3">
      <label class="flex min-w-55 flex-1 flex-col gap-1 text-xs text-secondary">
        {{ t('dashboard.mediaDrive.webdavPassword') }}
        <input
          v-model="webdavPassword"
          type="password"
          autocomplete="off"
          :placeholder="t('dashboard.mediaDrive.passwordHint')"
          class="rounded-md border border-border bg-bg-secondary px-3 py-2 text-sm text-main"
        />
      </label>
      <CustomButton
        type="primary"
        :icon="Play"
        :text="t('dashboard.mediaDrive.start')"
        :disabled="isBusy || isReady"
        @click="start"
      />
      <CustomButton
        type="custom"
        class="bg-danger/80! hover:bg-danger!"
        text-class="text-white"
        icon-class="text-white"
        :icon="Square"
        :text="t('dashboard.mediaDrive.stop')"
        :disabled="isBusy || !health"
        @click="stop"
      />
      <CustomButton
        type="secondary"
        :icon="ExternalLink"
        :text="t('dashboard.mediaDrive.openVlc')"
        :disabled="!isReady"
        @click="openVlc"
      />
    </div>

    <div v-if="error" class="rounded-md border border-danger/40 bg-danger/5 p-3 text-xs">
      <div class="font-semibold text-danger">{{ error.message }}</div>
      <div class="mt-1 text-secondary">{{ t('dashboard.mediaDrive.suggestion') }}: {{ error.suggestion }}</div>
    </div>
    <div
      v-else-if="isReady"
      class="flex items-center gap-2 rounded-md border border-success/40 bg-success/5 p-3 text-sm font-medium text-success"
    >
      <CircleCheck :size="17" />
      {{ t('dashboard.mediaDrive.ready') }}
      <span class="text-xs font-normal text-secondary">{{
        t('dashboard.mediaDrive.vlcHint', { drive: driveLetter })
      }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { CircleAlert, CircleCheck, ExternalLink, HardDrive, Play, Settings2, Square } from 'lucide-vue-next'
import { storeToRefs } from 'pinia'
import { computed, onMounted, ref } from 'vue'

import { useTranslation } from '../../composables/useI18n'
import { useMediaDriveStore } from '../../stores/mediaDrive'
import CustomButton from '../common/CustomButton.vue'

const { t } = useTranslation()
const mediaDriveStore = useMediaDriveStore()
const { state, environment, health, error, config, driveLetter, isReady, isBusy, stopped, stopping } =
  storeToRefs(mediaDriveStore)
const webdavPassword = ref('')
const driveInput = ref(driveLetter.value)
const autoStartInput = ref(config.value.autoStart)

const authReady = computed(() => health.value?.auth.ready ?? false)
const webdavReady = computed(() => health.value?.webdav.ready ?? false)
const mountReady = computed(() => health.value?.mount.ready ?? false)
const statusLabel = computed(() => {
  if (stopping.value) return t('dashboard.mediaDrive.stopping')
  switch (state.value) {
    case 'RUNNING':
      return t('dashboard.mediaDrive.ready')
    case 'CHECKING_ENV':
      return t('dashboard.mediaDrive.checking')
    case 'STARTING':
      return t('dashboard.mediaDrive.starting')
    case 'WAITING_AUTH':
      return t('dashboard.mediaDrive.waitingAuth')
    case 'DEGRADED':
      return t('dashboard.mediaDrive.degraded')
    case 'ERROR':
      return t('dashboard.mediaDrive.error')
    default:
      return t('dashboard.mediaDrive.readyToStart')
  }
})
const statusClass = computed(() => {
  if (state.value === 'RUNNING') return 'bg-success/70 text-white'
  if (state.value === 'ERROR' || state.value === 'DEGRADED') return 'bg-danger/50 text-white'
  return 'bg-accent/15 text-main'
})

const saveSetupAndStart = async () => {
  if (!mediaDriveStore.saveUserConfig(driveInput.value, autoStartInput.value)) return
  await mediaDriveStore.start(webdavPassword.value)
  webdavPassword.value = ''
}

const start = async () => {
  if (mediaDriveStore.saveUserConfig(driveInput.value, autoStartInput.value)) {
    await mediaDriveStore.start(webdavPassword.value)
    webdavPassword.value = ''
  }
}

const stop = () => mediaDriveStore.stop()
const openVlc = () => mediaDriveStore.openVlc()

onMounted(() => {
  if (state.value === 'APP_STARTING') void mediaDriveStore.initialize()
})
</script>
