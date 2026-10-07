<script setup lang="ts">
import { computed, ref } from 'vue'
import { urlsApi } from '../services/urls'
import { ApiError, describeError } from '../services/api'
import type { UrlItem } from '../types'
import { useAuth } from '../composables/useAuth'
import { useToast } from '../composables/useToast'
import BaseButton from './BaseButton.vue'
import UrlResultCard from './UrlResultCard.vue'

const { isLoggedIn } = useAuth()
const toast = useToast()

const targetUrl = ref('')
const alias = ref('')
const expiration = ref<'none' | '1h' | '1d' | '7d' | '30d' | 'custom'>('none')
const customDate = ref('')
const showOptions = ref(false)
const submitting = ref(false)
const created = ref<UrlItem | null>(null)
const fieldError = ref('')

const expirationOptions = computed(() => [
  { value: 'none', label: 'No expiration' },
  { value: '1h', label: '1 hour' },
  { value: '1d', label: '1 day' },
  { value: '7d', label: '7 days' },
  { value: '30d', label: '30 days' },
  { value: 'custom', label: 'Custom date' },
])

function normalizeTarget(raw: string): string {
  const trimmed = raw.trim()
  if (!trimmed) return trimmed
  if (!/^[a-zA-Z][a-zA-Z0-9+.-]*:/.test(trimmed)) return `https://${trimmed}`
  return trimmed
}

function computeExpiresAt(): string | undefined {
  const now = Date.now()
  const hours: Record<string, number | undefined> = { '1h': 1, '1d': 24, '7d': 24 * 7, '30d': 24 * 30 }
  const h = hours[expiration.value]
  if (h !== undefined) return new Date(now + h * 3600_000).toISOString()
  if (expiration.value === 'custom') {
    if (!customDate.value) return undefined
    const d = new Date(customDate.value)
    return Number.isNaN(d.getTime()) ? undefined : d.toISOString()
  }
  return undefined
}

const emit = defineEmits<{ created: [url: UrlItem] }>()

function resetForm(): void {
  targetUrl.value = ''
  alias.value = ''
  expiration.value = 'none'
  customDate.value = ''
  fieldError.value = ''
}

async function submit(): Promise<void> {
  fieldError.value = ''
  const target = normalizeTarget(targetUrl.value)
  if (!target) {
    fieldError.value = 'Please enter a URL to shorten.'
    return
  }
  submitting.value = true
  try {
    const result = await urlsApi.create({
      target_url: target,
      alias: alias.value.trim() || undefined,
      expires_at: computeExpiresAt(),
    })
    created.value = result
    resetForm()
    showOptions.value = false
    emit('created', result)
  } catch (e) {
    if (e instanceof ApiError && e.fields && Object.keys(e.fields).length > 0) {
      fieldError.value = Object.values(e.fields).join(' ')
    } else {
      const msg = describeError(e)
      fieldError.value = msg
      toast.error(msg)
    }
  } finally {
    submitting.value = false
  }
}

function createAnother(): void {
  created.value = null
}

defineExpose({ created })
</script>

<template>
  <form class="create-form" novalidate @submit.prevent="submit">
    <div class="row">
      <label class="sr-only" for="target-url">Long URL</label>
      <input
        id="target-url"
        v-model="targetUrl"
        class="input grow"
        type="url"
        inputmode="url"
        autocapitalize="off"
        autocomplete="off"
        spellcheck="false"
        placeholder="https://example.com/your/very/long/link"
        :aria-invalid="!!fieldError || undefined"
        :aria-describedby="fieldError ? 'create-error' : undefined"
      />
      <BaseButton type="submit" class="row-btn" :loading="submitting" block>
        {{ submitting ? 'Shortening…' : 'Shorten URL' }}
      </BaseButton>
    </div>

    <p v-if="fieldError" id="create-error" class="field-error" role="alert">{{ fieldError }}</p>

    <button type="button" class="options-toggle" :aria-expanded="showOptions" @click="showOptions = !showOptions">
      Options
      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" :style="{ transform: showOptions ? 'rotate(180deg)' : 'none' }" aria-hidden="true">
        <path d="M6 9l6 6 6-6" />
      </svg>
    </button>

    <div v-if="showOptions" class="options">
      <div class="option-field">
        <label class="field-label" for="alias">Custom alias</label>
        <template v-if="isLoggedIn">
          <input id="alias" v-model="alias" class="input" type="text" autocomplete="off" spellcheck="false" placeholder="my-link" />
        </template>
        <template v-else>
          <input class="input" type="text" disabled aria-disabled="true" placeholder="Log in to use a custom alias" />
          <p class="option-hint">
            <RouterLink to="/login">Log in to use a custom alias</RouterLink>
          </p>
        </template>
      </div>
      <div class="option-field">
        <label class="field-label" for="expiration">Expiration</label>
        <select id="expiration" v-model="expiration" class="select">
          <option v-for="opt in expirationOptions" :key="opt.value" :value="opt.value">{{ opt.label }}</option>
        </select>
        <label v-if="expiration === 'custom'" class="field-label" for="custom-date">Expiry date &amp; time</label>
        <input v-if="expiration === 'custom'" id="custom-date" v-model="customDate" class="input" type="datetime-local" />
      </div>
    </div>

    <UrlResultCard v-if="created" :url="created" @create-another="createAnother" />
  </form>
</template>

<style scoped>
.create-form {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}
.row {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}
@media (min-width: 768px) {
  .row {
    flex-direction: row;
    align-items: stretch;
  }
  .grow {
    flex: 1;
  }
  .row-btn {
    white-space: nowrap;
  }
  /* block button must not steal width from the input in a flex row */
  .row-btn.btn-block {
    width: auto;
  }
}
.options-toggle {
  align-self: flex-start;
  display: inline-flex;
  align-items: center;
  gap: var(--space-1);
  min-height: var(--tap-min);
  border: none;
  background: none;
  font: inherit;
  font-size: var(--text-sm);
  font-weight: var(--weight-medium);
  color: var(--color-muted);
  cursor: pointer;
  padding: 0 var(--space-2);
  border-radius: var(--radius-sm);
}
.options-toggle:hover {
  color: var(--color-text);
}
.options {
  display: grid;
  gap: var(--space-4);
  background: var(--color-surface-muted);
  border-radius: var(--radius-md);
  padding: var(--space-4);
}
@media (min-width: 768px) {
  .options {
    grid-template-columns: 1fr 1fr;
  }
}
.option-hint {
  font-size: var(--text-xs);
  color: var(--color-muted);
  margin-top: var(--space-1);
}
</style>
