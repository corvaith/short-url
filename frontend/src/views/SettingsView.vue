<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuth } from '../composables/useAuth'
import { describeError } from '../services/api'
import { useToast } from '../composables/useToast'
import { usePageMeta } from '../composables/usePageMeta'
import FormField from '../components/FormField.vue'
import BaseButton from '../components/BaseButton.vue'

usePageMeta('Settings')
const router = useRouter()
const toast = useToast()
const { user, logout } = useAuth()

const currentPassword = ref('')
const newPassword = ref('')
const saving = ref(false)
const passwordError = ref('')

async function changePassword(): Promise<void> {
  passwordError.value = ''
  saving.value = true
  try {
    await import('../services/auth').then(({ authApi }) =>
      authApi.changePassword(currentPassword.value, newPassword.value),
    )
    currentPassword.value = ''
    newPassword.value = ''
    toast.success('Password updated')
  } catch (e) {
    passwordError.value = describeError(e)
  } finally {
    saving.value = false
  }
}

async function onLogout(): Promise<void> {
  await logout()
  await router.push('/login')
}
</script>

<template>
  <div>
    <h1 class="page-title" tabindex="-1">Settings</h1>

    <section class="card setting-card">
      <h2 class="card-title">Account</h2>
      <div class="field">
        <label class="field-label" for="email">Email</label>
        <input
          id="email"
          class="input"
          type="email"
          :value="user?.email ?? ''"
          readonly
          aria-readonly="true"
        />
      </div>
    </section>

    <section class="card setting-card">
      <h2 class="card-title">Change password</h2>
      <form class="stack" novalidate @submit.prevent="changePassword">
        <FormField
          v-model="currentPassword"
          label="Current password"
          type="password"
          autocomplete="current-password"
          required
        />
        <FormField
          v-model="newPassword"
          label="New password"
          type="password"
          autocomplete="new-password"
          required
          hint="At least 8 characters."
        />
        <p v-if="passwordError" class="field-error" role="alert">{{ passwordError }}</p>
        <BaseButton type="submit" :loading="saving">Update password</BaseButton>
      </form>
    </section>

    <section class="card setting-card">
      <h2 class="card-title">Session</h2>
      <p class="muted-text">Signed in as {{ user?.email }}.</p>
      <BaseButton variant="danger" @click="onLogout">Log out</BaseButton>
    </section>
  </div>
</template>

<style scoped>
.page-title {
  font-size: var(--text-2xl);
  font-weight: var(--weight-semibold);
  margin-bottom: var(--space-6);
}
.page-title:focus {
  outline: none;
}
.setting-card {
  margin-bottom: var(--space-4);
}
.card-title {
  font-size: var(--text-lg);
  font-weight: var(--weight-semibold);
}
.stack {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
  margin-top: var(--space-4);
  max-width: 24rem;
}
.field {
  margin-top: var(--space-3);
  max-width: 24rem;
}
.muted-text {
  color: var(--color-muted);
  font-size: var(--text-sm);
  margin: var(--space-2) 0 var(--space-4);
}
</style>
