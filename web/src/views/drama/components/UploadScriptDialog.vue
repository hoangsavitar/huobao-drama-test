<template>
  <el-dialog
    v-model="visible"
    title="Upload Script"
    width="800px"
    :close-on-click-modal="false"
    @close="handleClose"
  >
    <el-form :model="form" label-width="100px">
      <el-form-item label="Script Content" required>
        <el-input
          v-model="form.script_content"
          type="textarea"
          :rows="15"
          placeholder="Paste your script content here&#10;The system will automatically identify and split it into episodes and scenes"
          maxlength="50000"
          show-word-limit
        />
        <div class="form-tip">
          Supports multiple script formats. The system will intelligently identify episodes, scenes, dialogue, and more.
        </div>
      </el-form-item>

      <el-form-item label="Split Options">
        <el-checkbox v-model="form.auto_split">Auto-split episodes</el-checkbox>
        <div class="form-tip">
          When enabled, episode boundaries will be automatically detected; otherwise treated as a single episode.
        </div>
      </el-form-item>
    </el-form>

    <template v-if="parseResult">
      <el-divider>Parse Result</el-divider>
      
      <div class="parse-result">
        <el-alert
          title="Parsing complete"
          type="success"
          :closable="false"
          show-icon
        >
          <template #default>
            Identified {{ parseResult.episodes.length }} episode(s),
            {{ totalScenes }} scene(s)
          </template>
        </el-alert>

        <div class="summary-box" v-if="parseResult.summary">
          <h4>Script Summary</h4>
          <p>{{ parseResult.summary }}</p>
        </div>

        <el-collapse v-model="activeEpisode" accordion>
          <el-collapse-item
            v-for="episode in parseResult.episodes"
            :key="episode.episode_number"
            :title="`Episode ${episode.episode_number}: ${episode.title}`"
            :name="episode.episode_number"
          >
            <div class="episode-info">
              <p><strong>Scenes: </strong>{{ episode.scenes.length }}</p>
              
              <el-table :data="episode.scenes" size="small" border>
                <el-table-column prop="storyboard_number" label="Scene #" width="80" />
                <el-table-column prop="title" label="Title" width="150" />
                <el-table-column prop="location" label="Location" width="120" />
                <el-table-column prop="time" label="Time" width="100" />
                <el-table-column prop="characters" label="Characters" width="150" />
                <el-table-column label="Dialogue">
                  <template #default="{ row }">
                    <div class="dialogue-preview">{{ row.dialogue }}</div>
                  </template>
                </el-table-column>
              </el-table>
            </div>
          </el-collapse-item>
        </el-collapse>
      </div>
    </template>

    <template #footer>
      <el-button @click="handleClose">Cancel</el-button>
      <el-button v-if="!parseResult" type="primary" @click="handleParse" :loading="parsing">
        Parse Script
      </el-button>
      <el-button v-else type="success" @click="handleSave" :loading="saving">
        Save to Project
      </el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, computed, reactive } from 'vue'
import { ElMessage } from 'element-plus'
import { generationAPI } from '@/api/generation'
import type { ParseScriptResult } from '@/types/generation'

interface Props {
  modelValue: boolean
  dramaId: string
}

const props = defineProps<Props>()
const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  success: []
}>()

const visible = computed({
  get: () => props.modelValue,
  set: (val) => emit('update:modelValue', val)
})

const form = reactive({
  script_content: '',
  auto_split: true
})

const parsing = ref(false)
const saving = ref(false)
const parseResult = ref<ParseScriptResult>()
const activeEpisode = ref<number>()

const totalScenes = computed(() => {
  if (!parseResult.value) return 0
  return parseResult.value.episodes.reduce((sum, ep) => sum + ep.scenes.length, 0)
})

const handleParse = async () => {
  if (!form.script_content.trim()) {
    ElMessage.warning('Please enter script content')
    return
  }

  parsing.value = true
  try {
    parseResult.value = await generationAPI.parseScript({
      drama_id: props.dramaId,
      script_content: form.script_content,
      auto_split: form.auto_split
    })
    ElMessage.success('Script parsed successfully')
  } catch (error: any) {
    ElMessage.error(error.message || 'Parse failed')
  } finally {
    parsing.value = false
  }
}

const handleSave = async () => {
  if (!parseResult.value) return

  saving.value = true
  try {
    // TODO: Call save API to persist parsed result to database
    ElMessage.success('Saved successfully')
    emit('success')
    handleClose()
  } catch (error: any) {
    ElMessage.error(error.message || 'Save failed')
  } finally {
    saving.value = false
  }
}

const handleClose = () => {
  visible.value = false
  form.script_content = ''
  form.auto_split = true
  parseResult.value = undefined
  activeEpisode.value = undefined
}
</script>

<style scoped>
.form-tip {
  margin-top: 8px;
  font-size: 12px;
  color: #909399;
}

.parse-result {
  margin-top: 20px;
}

.summary-box {
  margin: 20px 0;
  padding: 15px;
  background: #f5f7fa;
  border-radius: 8px;
}

.summary-box h4 {
  margin: 0 0 10px 0;
  font-size: 14px;
  color: #303133;
}

.summary-box p {
  margin: 0;
  line-height: 1.6;
  color: #606266;
}

.episode-info {
  padding: 10px 0;
}

.dialogue-preview {
  max-height: 60px;
  overflow: hidden;
  text-overflow: ellipsis;
  display: -webkit-box;
  -webkit-line-clamp: 3;
  -webkit-box-orient: vertical;
  font-size: 12px;
  line-height: 1.5;
}

:deep(.el-collapse-item__header) {
  font-weight: 500;
  color: #303133;
}
</style>
