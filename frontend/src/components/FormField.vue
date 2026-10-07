<script setup lang="ts">
import { useId } from 'vue'

withDefaults(
  defineProps<{
    label: string
    type?: string
    modelValue?: string
    placeholder?: string
    error?: string
    hint?: string
    required?: boolean
    autocomplete?: string
    inputmode?: 'text' | 'search' | 'none' | 'email' | 'tel' | 'url' | 'numeric' | 'decimal'
    autocapitalize?: string
    spellcheck?: boolean
    disabled?: boolean
  }>(),
  {
    type: 'text',
    modelValue: '',
    placeholder: '',
    error: '',
    hint: '',
    required: false,
    autocomplete: 'off',
    inputmode: undefined,
    autocapitalize: undefined,
    spellcheck: undefined,
    disabled: false,
  },
)

const emit = defineEmits<{ 'update:modelValue': [value: string] }>()
const id = useId()
const errorId = `${id}-error`
const hintId = `${id}-hint`
</script>

<template>
  <div class="form-field">
    <label class="field-label" :for="id">{{ label }}</label>
    <input
      :id="id"
      class="input"
      :class="{ invalid: !!error }"
      :type="type"
      :value="modelValue"
      :placeholder="placeholder"
      :required="required"
      :autocomplete="autocomplete"
      :inputmode="inputmode"
      :autocapitalize="autocapitalize"
      :spellcheck="spellcheck"
      :disabled="disabled"
      :aria-invalid="!!error || undefined"
      :aria-describedby="error ? errorId : hint ? hintId : undefined"
      @input="emit('update:modelValue', ($event.target as HTMLInputElement).value)"
    />
    <p v-if="error" :id="errorId" class="field-error">{{ error }}</p>
    <p v-else-if="hint" :id="hintId" class="field-hint">{{ hint }}</p>
  </div>
</template>

<style scoped>
.field-hint {
  color: var(--color-muted);
  font-size: var(--text-xs);
  margin-top: var(--space-1);
}
</style>
