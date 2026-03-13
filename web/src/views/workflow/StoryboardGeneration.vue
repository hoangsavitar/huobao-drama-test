<template>
  <div class="storyboard-generation">
    <el-empty description="Storyboard breakdown feature coming soon" />
  </div>
</template>

<script setup lang="ts">
defineProps<{
  dramaId: string
  episodeId: string
}>()

defineEmits<{
  storyboardGenerated: []
}>()
</script>

<style scoped lang="scss">
.storyboard-generation {
  padding: 24px;
}
</style>
