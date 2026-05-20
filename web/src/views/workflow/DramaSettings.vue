<template>
  <div class="drama-settings-container">
    <el-page-header @back="goBack" title="Back to Project">
      <template #content>
        <h2>Project Settings</h2>
      </template>
    </el-page-header>

    <el-card shadow="never" class="main-card">
      <el-tabs v-model="activeTab">
        <el-tab-pane label="Basic Info" name="basic">
          <el-form :model="form" label-width="100px" style="max-width: 600px">
            <el-form-item label="Title">
              <el-input v-model="form.title" />
            </el-form-item>
            <el-form-item label="Description">
              <el-input v-model="form.description" type="textarea" :rows="4" />
            </el-form-item>
            <el-form-item label="Genre">
              <el-select v-model="form.genre">
                <el-option label="Urban" value="都市" />
                <el-option label="Historical" value="古装" />
                <el-option label="Mystery" value="悬疑" />
                <el-option label="Romance" value="爱情" />
                <el-option label="Comedy" value="喜剧" />
              </el-select>
            </el-form-item>
            <el-form-item label="Status">
              <el-select v-model="form.status">
                <el-option label="Draft" value="draft" />
                <el-option label="Planning" value="planning" />
                <el-option label="In Production" value="production" />
                <el-option label="Completed" value="completed" />
                <el-option label="Archived" value="archived" />
              </el-select>
            </el-form-item>
            <el-form-item>
              <el-button type="primary" @click="saveSettings">Save Settings</el-button>
            </el-form-item>
          </el-form>
        </el-tab-pane>

        <el-tab-pane label="Danger Zone" name="danger">
          <el-alert
            title="Warning"
            type="warning"
            description="The following operations cannot be undone. Please proceed with caution."
            :closable="false"
            show-icon
          />
          <div class="danger-zone">
            <el-button type="danger" @click="deleteProject">Delete Project</el-button>
          </div>
        </el-tab-pane>
      </el-tabs>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { dramaAPI } from '@/api/drama'

const route = useRoute()
const router = useRouter()
const dramaId = route.params.id as string

const activeTab = ref('basic')
const form = reactive({
  title: '',
  description: '',
  genre: '',
  status: 'draft' as any
})

const goBack = () => {
  router.push(`/dramas/${dramaId}`)
}

const saveSettings = async () => {
  try {
    await dramaAPI.update(dramaId, form)
    ElMessage.success('Settings saved successfully')
  } catch (error: any) {
    ElMessage.error(error.message || 'Save failed')
  }
}

const deleteProject = async () => {
  try {
    await ElMessageBox.confirm(
      'Are you sure to delete this project? This cannot be undone!',
      'Warning',
      {
        confirmButtonText: 'Confirm Delete',
        cancelButtonText: 'Cancel',
        type: 'warning',
      }
    )
    
    await dramaAPI.delete(dramaId)
    ElMessage.success('Project deleted')
    router.push('/dramas')
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error(error.message || 'Delete failed')
    }
  }
}

onMounted(async () => {
  try {
    const drama = await dramaAPI.get(dramaId)
    Object.assign(form, drama)
  } catch (error: any) {
    ElMessage.error(error.message || 'Load failed')
  }
})
</script>

<style scoped>
.drama-settings-container {
  padding: 24px;
  max-width: 1200px;
  margin: 0 auto;
}

.main-card {
  margin-top: 20px;
}

.danger-zone {
  margin-top: 20px;
  padding: 20px;
  text-align: center;
}
</style>
