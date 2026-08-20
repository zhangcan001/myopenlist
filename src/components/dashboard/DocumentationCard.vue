<template>
  <div class="flex flex-col gap-4 w-full justify-center p-4">
    <div class="flex flex-col gap-5">
      <div class="flex flex-row gap-2">
        <div class="flex-1 border border-border-secondary rounded-md p-4 bg-bg-secondary flex flex-col gap-4">
          <div class="flex justify-between items-start">
            <div class="flex-1 flex flex-col gap-1">
              <h4 class="text-sm font-semibold text-main">{{ t('dashboard.documentation.openlist') }}</h4>
              <p class="text-secondary font-medium leading-[1.4] text-xs">
                {{ t('dashboard.documentation.openlistDesc') }}
              </p>
            </div>
            <div
              class="flex items-center justify-center w-8 h-8 border-none rounded-md shrink-0 bg-[linear-gradient(135deg,rgba(99,102,241,1),rgba(139,92,246,1))] text-white"
            >
              <BookOpen :size="20" />
            </div>
          </div>
          <div class="flex gap-2 flex-wrap">
            <CustomButton
              type="primary"
              :icon="ExternalLink"
              class="flex-1!"
              :text="t('dashboard.documentation.openDocs')"
              @click="openLink(urlMap.openlistDocs)"
            />
            <CustomButton
              type="secondary"
              :icon="Github"
              class="flex-1!"
              :text="t('dashboard.documentation.github')"
              @click="openLink(urlMap.openlistGitHub)"
            />
          </div>
        </div>

        <div class="flex-1 border border-border-secondary p-4 bg-bg-secondary rounded-md flex flex-col gap-4">
          <div class="flex justify-between items-start">
            <div class="flex-1 flex flex-col gap-1">
              <h4 class="text-sm font-semibold text-main">{{ t('dashboard.documentation.rclone') }}</h4>
              <p class="text-secondary font-medium leading-[1.4] text-xs">
                {{ t('dashboard.documentation.rcloneDesc') }}
              </p>
            </div>
            <div
              class="flex items-center justify-center w-8 h-8 border-none rounded-md shrink-0 bg-[linear-gradient(135deg,rgba(34,197,94,1),rgba(59,130,246,1))] text-white"
            >
              <Cloud :size="20" />
            </div>
          </div>
          <div class="flex gap-2 flex-wrap">
            <CustomButton
              type="primary"
              :icon="ExternalLink"
              class="flex-1!"
              :text="t('dashboard.documentation.openDocs')"
              @click="openLink(urlMap.rcloneDocs)"
            />
            <CustomButton
              type="secondary"
              :icon="Github"
              class="flex-1!"
              :text="t('dashboard.documentation.github')"
              @click="openLink(urlMap.rcloneGitHub)"
            />
          </div>
        </div>
      </div>

      <div class="rounded-md px-4 bg-bg-secondary">
        <div class="flex flex-wrap items-center justify-between gap-2">
          <CustomButton
            type="secondary"
            :icon="Code"
            :text="t('dashboard.documentation.apiDocs')"
            class="flex-1!"
            @click="openLink(urlMap.openlistAPIDocs)"
          />
          <CustomButton
            type="secondary"
            :icon="Terminal"
            :text="t('dashboard.documentation.commands')"
            class="flex-1!"
            @click="openLink(urlMap.rcloneCommands)"
          />
          <CustomButton
            type="secondary"
            :icon="HelpCircle"
            :text="t('dashboard.documentation.issues')"
            class="flex-1!"
            @click="openLink(urlMap.openlistIssues)"
          />
          <CustomButton
            type="secondary"
            :icon="MessageCircle"
            :text="t('dashboard.documentation.faq')"
            class="flex-1!"
            @click="openLink(urlMap.faq)"
          />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { BookOpen, Cloud, Code, ExternalLink, Github, HelpCircle, MessageCircle, Terminal } from 'lucide-vue-next'

import { createNewWindow } from '@/utils/common'
import { isMacOs } from '@/utils/constant'

import { TauriAPI } from '../../api/tauri'
import { useTranslation } from '../../composables/useI18n'
import { useAppStore } from '../../stores/app'
import CustomButton from '../common/CustomButton.vue'

const { t } = useTranslation()
const appStore = useAppStore()

const urlMap = {
  openlistAPIDocs: 'https://fox.oplist.org.cn/',
  rcloneCommands: 'https://rclone.org/commands/',
  openlistDocs: 'https://docs.oplist.org/',
  openlistGitHub: 'https://github.com/OpenListTeam/OpenList',
  openlistIssues: 'https://github.com/OpenListTeam/OpenList-desktop/issues',
  faq: 'https://doc.oplist.org/faq/howto',
  rcloneDocs: 'https://rclone.org/docs/',
  rcloneGitHub: 'https://github.com/rclone/rclone',
}

const openLink = async (url: string) => {
  if (appStore.settings.app.open_links_in_browser || isMacOs) {
    try {
      await TauriAPI.files.urlInBrowser(url)
    } catch (error) {
      console.error('Failed to open link:', error)
    }
    return
  }
  createNewWindow(url, `webview-${Date.now()}`, 'External Link')
}
</script>
