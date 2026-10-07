<script setup lang="ts">
import TopBar from '../components/TopBar.vue'
import BottomNav from '../components/BottomNav.vue'
import { useAuth } from '../composables/useAuth'
import { useBreakpoint } from '../composables/useBreakpoint'
import { computed } from 'vue'

const { isLoggedIn } = useAuth()
const { isMobile } = useBreakpoint()
const showBottomNav = computed(() => isLoggedIn.value && isMobile.value)
</script>

<template>
  <div class="layout" :class="{ 'has-bottomnav': showBottomNav }">
    <TopBar />
    <main id="main" class="container main-area">
      <RouterView />
    </main>
    <BottomNav v-if="showBottomNav" />
  </div>
</template>

<style scoped>
.layout {
  min-height: 100dvh;
  display: flex;
  flex-direction: column;
}
.main-area {
  flex: 1;
  padding-top: var(--space-6);
  padding-bottom: var(--space-8);
}
.layout.has-bottomnav .main-area {
  padding-bottom: calc(var(--bottomnav-h) + env(safe-area-inset-bottom) + var(--space-6));
}
</style>
