<script setup lang="ts">
import { useClipboard } from '../composables/useClipboard'
import { useToast } from '../composables/useToast'
import BaseButton from './BaseButton.vue'

const props = withDefaults(
  defineProps<{
    text: string
    label?: string
    variant?: 'primary' | 'secondary' | 'ghost'
    size?: 'md' | 'sm'
    block?: boolean
  }>(),
  {
    label: 'Copy',
    variant: 'secondary',
    size: 'md',
    block: false,
  },
)

const { copied, copy } = useClipboard()
const toast = useToast()

async function onClick(): Promise<void> {
  const ok = await copy(props.text)
  toast.info(ok ? 'Copied to clipboard' : 'Copy failed — select the text and copy manually')
}
</script>

<template>
  <BaseButton :variant="variant" :size="size" :block="block" @click="onClick">
    {{ copied ? 'Copied!' : label }}
  </BaseButton>
</template>
