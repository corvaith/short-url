<script setup lang="ts">
import { useId } from 'vue'

defineProps<{
  label: string
  modelValue: string
  options: { value: string; label: string }[]
  disabled?: boolean
}>()

const emit = defineEmits<{ 'update:modelValue': [value: string] }>()
const id = useId()
</script>

<template>
  <div>
    <label class="field-label" :for="id">{{ label }}</label>
    <select
      :id="id"
      class="select"
      :value="modelValue"
      :disabled="disabled"
      @change="emit('update:modelValue', ($event.target as HTMLSelectElement).value)"
    >
      <option v-for="opt in options" :key="opt.value" :value="opt.value">{{ opt.label }}</option>
    </select>
  </div>
</template>
