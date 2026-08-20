<script setup lang="ts">
import { ChevronDown } from 'lucide-vue-next'
import { computed, onMounted, ref } from 'vue'

import { useTranslation } from '../../composables/useI18n'

const { currentLocale, switchLanguage } = useTranslation()
const isOpen = ref(false)

const languages = [
  { code: 'zh', name: '中文', flag: '🇨🇳' },
  { code: 'en', name: 'English', flag: '🇺🇸' },
]

const currentLanguage = computed(() => languages.find(lang => lang.code === currentLocale.value))

const handleLanguageChange = (langCode: string) => {
  switchLanguage(langCode)
  isOpen.value = false
}

const toggleDropdown = () => {
  isOpen.value = !isOpen.value
}

const dropdownRef = ref<HTMLElement>()

onMounted(() => {
  document.addEventListener('click', e => {
    if (dropdownRef.value && !dropdownRef.value.contains(e.target as Node)) {
      isOpen.value = false
    }
  })
})
</script>

<template>
  <div ref="dropdownRef" class="language-switcher relative">
    <button class="language-button" @click="toggleDropdown">
      <span class="language-label">{{ currentLanguage?.name }}</span>
      <ChevronDown :size="12" :class="{ flipped: isOpen }" />
    </button>

    <div v-if="isOpen" class="language-dropdown">
      <div
        v-for="language in languages"
        :key="language.code"
        class="language-option"
        :class="{ active: language.code === currentLocale }"
        @click="handleLanguageChange(language.code)"
      >
        <span class="language-flag">{{ language.flag }}</span>
        <span class="language-name">{{ language.name }}</span>
        <span v-if="language.code === currentLocale" class="language-check">✓</span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.language-switcher {
  position: relative;
}

.language-button {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 0.75rem;
  background: rgba(255, 255, 255, 0.1);
  border: 1px solid rgba(255, 255, 255, 0.2);
  border-radius: 6px;
  color: var(--color-text-primary);
  font-size: 0.875rem;
  cursor: pointer;
  min-width: 120px;
}

.language-button:hover {
  background: rgba(247, 218, 218, 0.15);
  border-color: rgba(131, 60, 60, 0.3);
}

.language-label {
  flex: 1;
  text-align: center;
}

.flipped {
  opacity: 0.7;
}

.language-dropdown {
  position: absolute;
  top: 100%;
  width: 100%;
  text-align: center;
  margin-top: 0.25rem;
  background: var(--color-background-primary);
  border: 1px solid var(--color-border);
  border-radius: 8px;
  box-shadow: var(--shadow-lg);
  overflow: hidden;
  z-index: 50;
  min-width: 50px;
}

.language-option {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.75rem;
  cursor: pointer;
  font-size: 0.875rem;
}

.language-option:hover {
  background: var(--color-background-secondary);
}

.language-option.active {
  background: var(--color-accent);
  color: white;
}

.language-flag {
  font-size: 1rem;
}

.language-name {
  flex: 1;
}

.language-check {
  color: currentColor;
  font-weight: 600;
}

@media (prefers-color-scheme: dark) {
  .language-button {
    background: rgba(0, 0, 0, 0.2);
    border-color: rgba(255, 255, 255, 0.1);
  }

  .language-button:hover {
    background: rgba(0, 0, 0, 0.3);
    border-color: rgba(255, 255, 255, 0.2);
  }
}
</style>
