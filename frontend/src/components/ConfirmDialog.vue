<script setup lang="ts">
import { ref, watch } from 'vue'
import BaseButton from './BaseButton.vue'

const props = defineProps<{
  open: boolean
  title: string
  message: string
  confirmLabel?: string
  danger?: boolean
}>()

const emit = defineEmits<{ cancel: []; confirm: [] }>()

const dialogRef = ref<HTMLDialogElement | null>(null)

watch(
  () => props.open,
  (open) => {
    const dialog = dialogRef.value
    if (!dialog) return
    if (open && !dialog.open) {
      dialog.showModal()
      dialog.querySelector<HTMLButtonElement>('.js-cancel')?.focus()
    } else if (!open && dialog.open) {
      dialog.close()
    }
  },
)

function onCancel(): void {
  emit('cancel')
  dialogRef.value?.close()
}

function onConfirm(): void {
  emit('confirm')
  dialogRef.value?.close()
}
</script>

<template>
  <dialog ref="dialogRef" class="dialog" @cancel.prevent="onCancel">
    <div class="dialog-body">
      <h2 class="dialog-title">{{ title }}</h2>
      <p class="dialog-msg">{{ message }}</p>
      <div class="dialog-actions">
        <BaseButton variant="secondary" class="js-cancel" @click="onCancel">Cancel</BaseButton>
        <BaseButton :variant="danger ? 'danger' : 'primary'" @click="onConfirm">
          {{ confirmLabel ?? 'Confirm' }}
        </BaseButton>
      </div>
    </div>
  </dialog>
</template>

<style scoped>
.dialog {
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  padding: 0;
  box-shadow: var(--shadow-md);
  max-width: 26rem;
  width: calc(100vw - 2 * var(--page-pad));
}
.dialog::backdrop {
  background: rgb(18 22 28 / 0.4);
}
.dialog-body {
  padding: var(--space-6);
}
.dialog-title {
  font-size: var(--text-lg);
}
.dialog-msg {
  color: var(--color-muted);
  margin-top: var(--space-2);
  font-size: var(--text-sm);
}
.dialog-actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-3);
  margin-top: var(--space-6);
}
</style>
