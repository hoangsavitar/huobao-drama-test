<template>
  <div class="script-edit-container">
    <el-page-header @back="goBack" title="Back">
      <template #content>
        <h2>Script Editor</h2>
      </template>
    </el-page-header>
    <p>Feature coming soon...</p>
  </div>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router'

const router = useRouter()

const goBack = () => {
  router.back()
}
</script>

<style scoped>
.script-edit-container {
  padding: 20px;
}
</style>
