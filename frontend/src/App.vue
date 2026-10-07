<script setup lang="ts">
import SkipLink from './components/SkipLink.vue'
import ToastHost from './components/ToastHost.vue'
</script>

<template>
  <SkipLink />
  <RouterView />
  <ToastHost />
</template>
