<script setup lang="ts">
withDefaults(defineProps<{ height?: string; width?: string }>(), { height: '1rem', width: '100%' })
</script>

<template>
  <div class="skeleton" :style="{ height, width }" aria-hidden="true" />
</template>
