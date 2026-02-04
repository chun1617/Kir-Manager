<script setup lang="ts">
import { CARD_STYLES } from '@/constants/settingsStyles'

/**
 * SettingsCard 組件 Props
 * @requirements 7.1, 7.2 - 卡片佈局規格
 */
interface Props {
  /** 卡片標題 */
  title?: string
}

defineProps<Props>()
</script>

<template>
  <div :class="CARD_STYLES.card">
    <h3 v-if="title" class="text-lg font-medium text-zinc-100 mb-4">
      {{ title }}
    </h3>
    <slot />
  </div>
</template>
