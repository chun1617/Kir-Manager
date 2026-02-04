<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { REFRESH_INTERVAL_STYLES } from '@/constants/refreshIntervalStyles'

const props = withDefaults(defineProps<{
  disabled?: boolean
  disabledReason?: string | null
}>(), {
  disabled: false,
  disabledReason: null,
})

const emit = defineEmits<{
  'add': []
}>()

const { t } = useI18n()

const buttonClass = computed(() => 
  props.disabled 
    ? REFRESH_INTERVAL_STYLES.addButton.disabled 
    : REFRESH_INTERVAL_STYLES.addButton.enabled
)

const tooltipText = computed(() => 
  props.disabled && props.disabledReason 
    ? t(props.disabledReason) 
    : ''
)

function handleClick() {
  if (!props.disabled) {
    emit('add')
  }
}
</script>

<template>
  <button
    type="button"
    :class="buttonClass"
    :disabled="disabled"
    :title="tooltipText"
    data-testid="add-rule-button"
    @click="handleClick"
  >
    + {{ t('refreshInterval.addRule') }}
  </button>
</template>
