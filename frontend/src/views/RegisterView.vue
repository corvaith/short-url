<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuth } from '../composables/useAuth'
import { describeError } from '../services/api'
import { usePageMeta } from '../composables/usePageMeta'
import FormField from '../components/FormField.vue'
import BaseButton from '../components/BaseButton.vue'

usePageMeta('Create account')
const router = useRouter()
const { register } = useAuth()

const email = ref('')
const password = ref('')
const loading = ref(false)
const error = ref('')

async function submit(): Promise<void> {
  error.value = ''
  loading.value = true
  try {
    await register(email.value.trim(), password.value)
    await router.push('/dashboard')
  } catch (e) {
    error.value = describeError(e)
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div>
    <h1 class="auth-title" tabindex="-1">Create account</h1>
    <p class="auth-sub">Sign up to manage and track your short links.</p>

    <div class="oauth-row">
      <a class="btn btn-oauth btn-github" href="/api/v1/auth/oauth/github/start?next=%2Fdashboard">
        <svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor" aria-hidden="true"><path d="M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82.64-.18 1.32-.27 2-.27s1.36.09 2 .27c1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8.01 8.01 0 0 0 16 8c0-4.42-3.58-8-8-8Z"/></svg>
        Continue with GitHub
      </a>
      <a class="btn btn-oauth btn-discord" href="/api/v1/auth/oauth/discord/start?next=%2Fdashboard">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true"><path d="M20.32 4.37a19.8 19.8 0 0 0-4.93-1.51 13.8 13.8 0 0 0-.64 1.28 18.3 18.3 0 0 0-5.5 0 12.6 12.6 0 0 0-.64-1.28c-1.71.29-3.37.8-4.93 1.51A20.3 20.3 0 0 0 .1 18.06a19.9 19.9 0 0 0 6.07 3.03c.49-.66.93-1.37 1.3-2.1a12.9 12.9 0 0 1-2.05-.98c.17-.12.34-.25.5-.38a14.2 14.2 0 0 0 12.16 0c.16.13.33.26.5.38-.65.39-1.34.72-2.05.98.37.73.8 1.44 1.3 2.1a19.8 19.8 0 0 0 6.07-3.03 20.2 20.2 0 0 0-3.58-13.69ZM8.02 15.33c-1.18 0-2.16-1.08-2.16-2.42 0-1.33.95-2.42 2.16-2.42 1.21 0 2.18 1.1 2.16 2.42 0 1.34-.95 2.42-2.16 2.42Zm7.96 0c-1.18 0-2.16-1.08-2.16-2.42 0-1.33.95-2.42 2.16-2.42 1.21 0 2.18 1.1 2.16 2.42 0 1.34-.95 2.42-2.16 2.42Z"/></svg>
        Continue with Discord
      </a>
    </div>

    <div class="divider" role="separator" aria-label="or"><span>or</span></div>

    <form class="auth-form" novalidate @submit.prevent="submit">
      <FormField v-model="email" label="Email" type="email" autocomplete="email" required />
      <FormField v-model="password" label="Password" type="password" autocomplete="new-password" required hint="At least 8 characters." />
      <p v-if="error" class="field-error" role="alert">{{ error }}</p>
      <BaseButton type="submit" block :loading="loading">Create account</BaseButton>
    </form>
    <p class="auth-alt">
      Already have an account? <RouterLink to="/login">Log in</RouterLink>
    </p>
  </div>
</template>

<style scoped>
.auth-title {
  font-size: var(--text-2xl);
  font-weight: var(--weight-semibold);
}
.oauth-row {
  display: grid;
  gap: var(--space-2);
  margin-top: var(--space-6);
}
.btn-oauth {
  width: 100%;
  justify-content: flex-start;
  gap: var(--space-3);
  font-weight: var(--weight-regular);
  font-size: 12.8px;
}
.btn-github {
  background: #24292F;
  color: #FFFFFF;
  border-color: #24292F;
}
.btn-github:hover:not(:disabled) {
  background: #3A424A;
}
.btn-discord {
  background: #5865F2;
  color: #FFFFFF;
  border-color: #5865F2;
}
.btn-discord:hover:not(:disabled) {
  background: #4752C4;
}
.divider {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  margin-top: var(--space-5);
  color: var(--color-faint);
  font-size: var(--text-xs);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}
.divider::before,
.divider::after {
  content: '';
  flex: 1;
  height: 1px;
  background: var(--color-border);
}
.auth-title:focus {
  outline: none;
}
.auth-sub {
  color: var(--color-muted);
  font-size: var(--text-sm);
  margin-top: var(--space-1);
}
.auth-form {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
  margin-top: var(--space-6);
}
.auth-alt {
  margin-top: var(--space-6);
  font-size: var(--text-sm);
  color: var(--color-muted);
  text-align: center;
}
</style>
