<script setup lang="ts">
/**
 * NumberFlow 數字動畫組件
 * @description 封裝 @number-flow/vue，提供統一的數字動畫效果
 * @see https://number-flow.barvian.me/vue
 */
import NumberFlow, { type Format } from '@number-flow/vue'

interface Props {
  /** 要顯示的數值 */
  value: number
  /** 數字格式化選項 */
  format?: Format
  /** 前綴文字 */
  prefix?: string
  /** 後綴文字 */
  suffix?: string
  /** 是否啟用動畫 */
  animated?: boolean
}

withDefaults(defineProps<Props>(), {
  animated: true,
})
</script>

<template>
  <NumberFlow
    :value="value"
    :format="format"
    :prefix="prefix"
    :suffix="suffix"
    :animated="animated"
  />
</template>
