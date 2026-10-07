<script setup lang="ts">
import { ref } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuth } from '../composables/useAuth'
import { describeError } from '../services/api'
import { usePageMeta } from '../composables/usePageMeta'
import FormField from '../components/FormField.vue'
import BaseButton from '../components/BaseButton.vue'

usePageMeta('Log in')
const router = useRouter()
const route = useRoute()
const { login } = useAuth()

const email = ref('')
const password = ref('')
const loading = ref(false)
const error = ref('')

async function submit(): Promise<void> {
  error.value = ''
  loading.value = true
  try {
    await login(email.value.trim(), password.value)
    const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : '/dashboard'
    await router.push(redirect)
  } catch (e) {
    error.value = describeError(e)
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div>
    <h1 class="auth-title" tabindex="-1">Log in</h1>
    <p class="auth-sub">Welcome back. Enter your details to continue.</p>
    <form class="auth-form" novalidate @submit.prevent="submit">
      <FormField v-model="email" label="Email" type="email" autocomplete="email" required />
      <FormField v-model="password" label="Password" type="password" autocomplete="current-password" required />
      <p v-if="error" class="field-error" role="alert">{{ error }}</p>
      <BaseButton type="submit" block :loading="loading">Log in</BaseButton>
    </form>
    <p class="auth-alt">
      Don't have an account? <RouterLink to="/register">Create one</RouterLink>
    </p>
  </div>
</template>

<style scoped>
.auth-title {
  font-size: var(--text-2xl);
  font-weight: var(--weight-semibold);
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
