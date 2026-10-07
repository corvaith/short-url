<script setup lang="ts">
import { useId } from 'vue'

withDefaults(
  defineProps<{
    modelValue?: string
    type?: string
    placeholder?: string
    invalid?: boolean
    disabled?: boolean
    autocomplete?: string
    inputmode?: 'text' | 'search' | 'none' | 'email' | 'tel' | 'url' | 'numeric' | 'decimal'
    spellcheck?: boolean
    ariaLabel?: string
  }>(),
  {
    modelValue: '',
    type: 'text',
    placeholder: '',
    invalid: false,
    disabled: false,
    autocomplete: 'off',
    inputmode: undefined,
    spellcheck: undefined,
    ariaLabel: undefined,
  },
)

const emit = defineEmits<{ 'update:modelValue': [value: string] }>()
const id = useId()
</script>

<template>
  <input
    :id="id"
    class="input"
    :class="{ invalid }"
    :type="type"
    :value="modelValue"
    :placeholder="placeholder"
    :autocomplete="autocomplete"
    :inputmode="inputmode"
    :spellcheck="spellcheck"
    :disabled="disabled"
    :aria-label="ariaLabel"
    :aria-invalid="invalid || undefined"
    @input="emit('update:modelValue', ($event.target as HTMLInputElement).value)"
  />
</template>
