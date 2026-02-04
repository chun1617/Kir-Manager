<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import SettingsCard from './SettingsCard.vue'

/**
 * BasicSettingsTab 組件 Props
 * @requirements 2.1, 2.2, 2.3, 2.4 - 基礎設定分頁內容
 */
interface Props {
  /** Kiro 安裝路徑 */
  kiroInstallPath: string
  /** Kiro 版本號 */
  kiroVersion: string
  /** 介面語言 */
  language: string
  /** 低餘額閾值 (0-1) */
  lowBalanceThreshold: number
  /** 是否正在偵測版本 */
  detectingVersion?: boolean
  /** 是否正在偵測路徑 */
  detectingPath?: boolean
}

withDefaults(defineProps<Props>(), {
  detectingVersion: false,
  detectingPath: false,
})

const emit = defineEmits<{
  (e: 'update:kiroInstallPath', value: string): void
  (e: 'update:kiroVersion', value: string): void
  (e: 'update:language', value: string): void
  (e: 'update:lowBalanceThreshold', value: number): void
  (e: 'detectVersion'): void
  (e: 'detectPath'): void
  (e: 'saveVersion'): void
  (e: 'savePath'): void
}>()

const { t } = useI18n()

const languages = [
  { value: 'zh-TW', label: '繁體中文' },
  { value: 'zh-CN', label: '简体中文' },
]
</script>

<template>
  <div class="space-y-4">
    <!-- Kiro 安裝路徑 -->
    <SettingsCard :title="t('settings.kiroInstallPath')">
      <p class="text-sm text-zinc-400 mb-3">{{ t('settings.kiroInstallPathDesc') }}</p>
      <div class="flex gap-2">
        <input
          type="text"
          :value="kiroInstallPath"
          :placeholder="t('settings.kiroInstallPathPlaceholder')"
          class="flex-1 bg-zinc-900 border border-zinc-700 rounded-lg px-3 py-2 text-sm text-zinc-100"
          @input="emit('update:kiroInstallPath', ($event.target as HTMLInputElement).value)"
          @blur="emit('savePath')"
        />
        <button
          class="px-4 py-2 bg-zinc-700 hover:bg-zinc-600 rounded-lg text-sm"
          :disabled="detectingPath"
          @click="emit('detectPath')"
        >
          {{ detectingPath ? '...' : t('settings.detectPath') }}
        </button>
      </div>
    </SettingsCard>

    <!-- Kiro 版本號 -->
    <SettingsCard :title="t('settings.kiroVersion')">
      <p class="text-sm text-zinc-400 mb-3">{{ t('settings.kiroVersionDesc') }}</p>
      <div class="flex gap-2">
        <input
          type="text"
          :value="kiroVersion"
          :placeholder="t('settings.kiroVersionPlaceholder')"
          class="flex-1 bg-zinc-900 border border-zinc-700 rounded-lg px-3 py-2 text-sm text-zinc-100"
          @input="emit('update:kiroVersion', ($event.target as HTMLInputElement).value)"
          @blur="emit('saveVersion')"
        />
        <button
          class="px-4 py-2 bg-zinc-700 hover:bg-zinc-600 rounded-lg text-sm"
          :disabled="detectingVersion"
          @click="emit('detectVersion')"
        >
          {{ detectingVersion ? '...' : t('settings.detectVersion') }}
        </button>
      </div>
    </SettingsCard>

    <!-- 介面語言 -->
    <SettingsCard :title="t('settings.language')">
      <div class="flex gap-2">
        <button
          v-for="lang in languages"
          :key="lang.value"
          :class="[
            'px-4 py-2 rounded-lg text-sm transition-colors',
            language === lang.value
              ? 'bg-zinc-700 text-zinc-100'
              : 'bg-zinc-900 text-zinc-400 hover:bg-zinc-800'
          ]"
          @click="emit('update:language', lang.value)"
        >
          {{ lang.label }}
        </button>
      </div>
    </SettingsCard>

    <!-- 低餘額閾值 -->
    <SettingsCard :title="t('settings.lowBalanceThreshold')">
      <p class="text-sm text-zinc-400 mb-3">{{ t('settings.lowBalanceThresholdDesc') }}</p>
      <div class="flex items-center gap-4">
        <input
          type="range"
          :value="lowBalanceThreshold * 100"
          min="5"
          max="50"
          step="5"
          class="flex-1"
          @input="emit('update:lowBalanceThreshold', Number(($event.target as HTMLInputElement).value) / 100)"
        />
        <span class="text-sm text-zinc-300 w-12 text-right">
          {{ Math.round(lowBalanceThreshold * 100) }}%
        </span>
      </div>
    </SettingsCard>
  </div>
</template>
