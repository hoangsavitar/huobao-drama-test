import { createI18n } from 'vue-i18n'
import zhCN from './zh-CN'
import enUS from './en-US'

// Get stored language from localStorage, default to English
const getStoredLanguage = (): string => {
  const stored = localStorage.getItem('language')
  if (stored) return stored
  
  // Auto-detect browser language
  const browserLang = navigator.language.toLowerCase()
  if (browserLang.startsWith('zh')) return 'zh-CN'
  return 'en-US'
}

const i18n = createI18n({
  legacy: false, // Use Composition API mode
  locale: getStoredLanguage(),
  fallbackLocale: 'en-US',
  messages: {
    'zh-CN': zhCN,
    'en-US': enUS
  }
})

export default i18n

// Export language switch function
export const setLanguage = (lang: string) => {
  i18n.global.locale.value = lang as any
  localStorage.setItem('language', lang)
}

export const getCurrentLanguage = () => {
  return i18n.global.locale.value
}
