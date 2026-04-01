<script setup lang="ts">
defineProps<{
  status: 'healthy' | 'warning' | 'critical'
}>()
</script>

<template>
  <span class="relative flex h-3 w-3">
    <span v-if="status !== 'healthy'" class="animate-ping absolute inline-flex h-full w-full rounded-full opacity-75"
          :class="status === 'warning' ? 'bg-yellow-400' : 'bg-red-400'"></span>
    <span class="relative inline-flex rounded-full h-3 w-3"
          :class="{
            'bg-green-500': status === 'healthy',
            'bg-yellow-500': status === 'warning',
            'bg-red-500': status === 'critical'
          }"></span>
  </span>
</template>
