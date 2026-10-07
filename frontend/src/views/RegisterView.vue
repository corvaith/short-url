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
