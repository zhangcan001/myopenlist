import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

export function useTranslation() {
  const { t, locale, availableLocales } = useI18n()

  const currentLocale = computed(() => locale.value)

  const switchLanguage = (lang: string) => {
    locale.value = lang
    localStorage.setItem('preferred-language', lang)
  }

  const getStoredLanguage = () => {
    return localStorage.getItem('preferred-language') || 'zh'
  }

  const isChineseLocale = computed(() => locale.value === 'zh')
  const isEnglishLocale = computed(() => locale.value === 'en')

  return {
    t,
    currentLocale,
    availableLocales,
    switchLanguage,
    getStoredLanguage,
    isChineseLocale,
    isEnglishLocale,
  }
}
