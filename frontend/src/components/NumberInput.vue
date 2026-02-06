<script setup lang="ts">
/**
 * NumberInput 數字輸入組件
 * @description 帶有加減按鈕的數字輸入框，使用 NumberFlow 動畫效果
 */
import { computed, onMounted, ref } from 'vue'
import NumberFlow, { useCanAnimate } from '@number-flow/vue'

// 檢測動畫支援
const canAnimate = useCanAnimate()

// 調試信息（開發時可見）
const debugInfo = ref<{
  canAnimate: boolean
  supportsMod: boolean
  supportsLinear: boolean
  supportsAtProperty: boolean
  chromiumVersion: string
} | null>(null)

// 調試：在控制台輸出動畫支援狀態
onMounted(() => {
  // 檢測各項功能支援
  const supportsMod = typeof CSS !== 'undefined' && CSS.supports && CSS.supports('line-height', 'mod(1,1)')
  const supportsLinear = (() => {
    try {
      document.createElement('div').animate({ opacity: 0 }, { easing: 'linear(0, 1)' })
      return true
    } catch (e) {
      return false
    }
  })()
  const supportsAtProperty = (() => {
    try {
      CSS.registerProperty({
        name: '--test-number-input-prop',
        syntax: '<number>',
        inherits: false,
        initialValue: '0'
      })
      return true
    } catch {
      return false
    }
  })()
  
  // 提取 Chromium 版本
  const ua = navigator.userAgent
  const chromiumMatch = ua.match(/Chrome\/(\d+)/)
  const chromiumVersion = chromiumMatch ? chromiumMatch[1] : 'unknown'
  
  debugInfo.value = {
    canAnimate: canAnimate.value,
    supportsMod,
    supportsLinear,
    supportsAtProperty,
    chromiumVersion
  }
  
  console.log('[NumberInput] Feature detection:', debugInfo.value)
  console.log('[NumberInput] UserAgent:', ua)
  
  // 如果不支援動畫，輸出警告
  if (!canAnimate.value) {
    console.warn('[NumberInput] Animation not supported! Required: CSS mod() (Chrome 125+), linear() easing, CSS.registerProperty')
    if (!supportsMod) {
      console.warn('[NumberInput] CSS mod() not supported. Current Chrome version:', chromiumVersion, '(requires 125+)')
    }
  }
})

interface Props {
  /** 當前數值 */
  modelValue: number
  /** 最小值 */
  min?: number
  /** 最大值 */
  max?: number
  /** 步進值 */
  step?: number
  /** 是否禁用 */
  disabled?: boolean
  /** 數字顯示區域最小寬度 (Tailwind class，如 'min-w-[3rem]') */
  minWidth?: string
}

const props = withDefaults(defineProps<Props>(), {
  min: 0,
  max: Infinity,
  step: 1,
  disabled: false,
  minWidth: 'min-w-[3rem]',
})

const emit = defineEmits<{
  'update:modelValue': [value: number]
}>()

const canDecrease = computed(() => !props.disabled && props.modelValue > props.min)
const canIncrease = computed(() => !props.disabled && props.modelValue < props.max)

function decrease() {
  if (canDecrease.value) {
    const newValue = Math.max(props.min, props.modelValue - props.step)
    emit('update:modelValue', newValue)
  }
}

function increase() {
  if (canIncrease.value) {
    const newValue = Math.min(props.max, props.modelValue + props.step)
    emit('update:modelValue', newValue)
  }
}
</script>

<template>
  <div 
    :class="[
      'inline-flex items-center rounded border transition-colors',
      disabled 
        ? 'border-zinc-700 bg-zinc-900/50 opacity-50' 
        : 'border-violet-500/50 bg-zinc-900 hover:border-violet-500'
    ]"
  >
    <!-- 減號按鈕 -->
    <button
      type="button"
      :disabled="!canDecrease"
      :class="[
        'px-2 py-1 text-sm font-bold transition-colors',
        canDecrease 
          ? 'text-zinc-300 hover:text-white' 
          : 'text-zinc-600 cursor-not-allowed'
      ]"
      @click="decrease"
    >
      −
    </button>
    
    <!-- 數字顯示區域 -->
    <div :class="['px-1.5 py-1 text-center', minWidth]">
      <NumberFlow 
        :value="modelValue" 
        class="text-sm font-semibold text-white tabular-nums"
      />
    </div>
    
    <!-- 加號按鈕 -->
    <button
      type="button"
      :disabled="!canIncrease"
      :class="[
        'px-2 py-1 text-sm font-bold transition-colors',
        canIncrease 
          ? 'text-zinc-300 hover:text-white' 
          : 'text-zinc-600 cursor-not-allowed'
      ]"
      @click="increase"
    >
      +
    </button>
  </div>
</template>
