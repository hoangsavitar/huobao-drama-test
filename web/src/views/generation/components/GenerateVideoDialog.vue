<template>
  <el-dialog
    v-model="visible"
    title="AI Video Generation"
    width="700px"
    :close-on-click-modal="false"
    @close="handleClose"
  >
    <el-form :model="form" :rules="rules" ref="formRef" label-width="120px">
      <el-form-item label="Script" prop="drama_id">
        <el-select v-model="form.drama_id" placeholder="Select script" @change="onDramaChange">
          <el-option
            v-for="drama in dramas"
            :key="drama.id"
            :label="drama.title"
            :value="drama.id"
          />
        </el-select>
      </el-form-item>

      <el-form-item label="Select Image" prop="image_gen_id">
        <el-select
          v-model="form.image_gen_id"
          placeholder="Select a generated image"
          clearable
          @change="onImageChange"
        >
          <el-option
            v-for="image in images"
            :key="image.id"
            :label="truncateText(image.prompt, 50)"
            :value="image.id"
          >
            <div class="image-option">
              <img v-if="image.image_url" :src="image.image_url" class="image-thumb" />
              <span>{{ truncateText(image.prompt, 40) }}</span>
            </div>
          </el-option>
        </el-select>
        <div class="form-tip">Or enter image URL directly</div>
      </el-form-item>

      <el-form-item label="Image URL" prop="image_url">
        <el-input
          v-model="form.image_url"
          placeholder="https://example.com/image.jpg"
          :disabled="!!form.image_gen_id"
        />
      </el-form-item>

      <el-form-item label="Video Prompt" prop="prompt">
        <el-input
          v-model="form.prompt"
          type="textarea"
          :rows="5"
          placeholder="Describe the action and camera movement&#10;e.g. Camera slowly zooms in, wind blowing through hair, cinematic lighting"
          maxlength="2000"
          show-word-limit
        />
      </el-form-item>

      <el-form-item label="AI Service">
        <el-select v-model="form.provider" placeholder="Select service">
          <el-option label="Doubao Video" value="doubao" />
          <el-option label="Runway" value="runway" />
          <el-option label="Pika" value="pika" />
        </el-select>
      </el-form-item>

      <el-form-item label="Duration">
        <el-slider
          v-model="form.duration"
          :min="3"
          :max="10"
          :marks="durationMarks"
          show-stops
        />
        <span class="slider-value">{{ form.duration }}s</span>
      </el-form-item>

      <el-form-item label="Aspect Ratio">
        <el-radio-group v-model="form.aspect_ratio">
          <el-radio label="16:9">16:9 (Landscape)</el-radio>
          <el-radio label="9:16">9:16 (Portrait)</el-radio>
          <el-radio label="1:1">1:1 (Square)</el-radio>
        </el-radio-group>
      </el-form-item>

      <el-collapse>
        <el-collapse-item title="Advanced Settings" name="advanced">
          <el-form-item label="Motion Level">
            <el-slider
              v-model="form.motion_level"
              :min="0"
              :max="100"
              :marks="motionMarks"
            />
            <span class="slider-value">{{ form.motion_level }}</span>
          </el-form-item>

          <el-form-item label="Camera Motion">
            <el-select v-model="form.camera_motion" placeholder="Select camera motion" clearable>
              <el-option label="Static" value="static" />
              <el-option label="Zoom In" value="zoom_in" />
              <el-option label="Zoom Out" value="zoom_out" />
              <el-option label="Pan Left" value="pan_left" />
              <el-option label="Pan Right" value="pan_right" />
              <el-option label="Tilt Up" value="tilt_up" />
              <el-option label="Tilt Down" value="tilt_down" />
              <el-option label="Orbit" value="orbit" />
            </el-select>
          </el-form-item>

          <el-form-item label="Style" v-if="form.provider === 'doubao'">
            <el-input v-model="form.style" placeholder="e.g. Cinematic, Anime style" />
          </el-form-item>

          <el-form-item label="Seed">
            <el-input-number v-model="form.seed" :min="-1" placeholder="Leave empty for random" />
            <span class="form-tip">Same seed reproduces the same video</span>
          </el-form-item>
        </el-collapse-item>
      </el-collapse>
    </el-form>

    <template #footer>
      <el-button @click="handleClose">Cancel</el-button>
      <el-button type="primary" :loading="generating" @click="handleGenerate">
        Generate Video
      </el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, reactive, computed, watch } from 'vue'
import { ElMessage, type FormInstance, type FormRules } from 'element-plus'
import { videoAPI } from '@/api/video'
import { imageAPI } from '@/api/image'
import { dramaAPI } from '@/api/drama'
import type { Drama } from '@/types/drama'
import type { ImageGeneration } from '@/types/image'
import type { GenerateVideoRequest } from '@/types/video'

interface Props {
  modelValue: boolean
  dramaId?: string
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

const formRef = ref<FormInstance>()
const generating = ref(false)
const dramas = ref<Drama[]>([])
const images = ref<ImageGeneration[]>([])

const form = reactive<GenerateVideoRequest & { image_gen_id?: number }>({
  drama_id: props.dramaId || '',
  image_gen_id: undefined,
  image_url: '',
  prompt: '',
  provider: 'doubao',
  duration: 5,
  aspect_ratio: '16:9',
  motion_level: 50,
  camera_motion: undefined,
  style: undefined,
  seed: undefined
})

const rules: FormRules = {
  drama_id: [
    { required: true, message: 'Please select a script', trigger: 'change' }
  ],
  prompt: [
    { required: true, message: 'Please enter a video prompt', trigger: 'blur' },
    { min: 5, message: 'Prompt must be at least 5 characters', trigger: 'blur' }
  ]
}

const durationMarks = {
  3: '3s',
  5: '5s',
  7: '7s',
  10: '10s'
}

const motionMarks = {
  0: 'Still',
  50: 'Medium',
  100: 'Dynamic'
}

watch(() => props.modelValue, async (val) => {
  if (val) {
    await loadDramas()
    if (props.dramaId) {
      form.drama_id = props.dramaId
      loadImages(props.dramaId)
      const drama = dramas.value.find(d => d.id === props.dramaId)
      if (drama?.aspect_ratio) {
        form.aspect_ratio = drama.aspect_ratio
      }
    }
  }
})

const loadDramas = async (): Promise<void> => {
  try {
    const result = await dramaAPI.list({ page: 1, page_size: 100 })
    dramas.value = result.items
  } catch (error: any) {
    console.error('Failed to load dramas:', error)
  }
}

const loadImages = async (dramaId: string) => {
  try {
    const result = await imageAPI.listImages({
      drama_id: dramaId,
      status: 'completed',
      page: 1,
      page_size: 100
    })
    images.value = result.items
  } catch (error: any) {
    console.error('Failed to load images:', error)
  }
}

const onDramaChange = (dramaId: string) => {
  form.image_gen_id = undefined
  form.image_url = ''
  images.value = []
  if (dramaId) {
    loadImages(dramaId)
    const drama = dramas.value.find(d => d.id === dramaId)
    if (drama?.aspect_ratio) {
      form.aspect_ratio = drama.aspect_ratio
    }
  }
}

const onImageChange = (imageGenId: number | undefined) => {
  if (!imageGenId) {
    form.image_url = ''
    return
  }
  
  const image = images.value.find(img => img.id === imageGenId)
  if (image && image.image_url) {
    form.image_url = image.image_url
    form.prompt = image.prompt
  }
}

const truncateText = (text: string, length: number) => {
  if (text.length <= length) return text
  return text.substring(0, length) + '...'
}

const handleGenerate = async () => {
  console.log('handleGenerate called')
  
  if (!formRef.value) {
    console.error('formRef is null')
    ElMessage.error('Form initialization failed, please refresh the page')
    return
  }

  try {
    const valid = await formRef.value.validate()
    console.log('Form validation result:', valid)
    
    if (!valid) {
      console.log('Form validation failed')
      return
    }

    generating.value = true
    console.log('Starting video generation...', form)
    
    try {
      if (form.image_gen_id) {
        console.log('Generating from image:', form.image_gen_id)
        await videoAPI.generateFromImage(form.image_gen_id)
      } else {
        const params: GenerateVideoRequest = {
          drama_id: form.drama_id,
          prompt: form.prompt,
          provider: form.provider
        }

        if (form.image_url && form.image_url.trim()) {
          params.image_url = form.image_url
          params.reference_mode = 'single'
        } else {
          params.reference_mode = 'none'
        }

        if (form.duration) params.duration = form.duration
        if (form.aspect_ratio) params.aspect_ratio = form.aspect_ratio
        if (form.motion_level !== undefined) params.motion_level = form.motion_level
        if (form.camera_motion) params.camera_motion = form.camera_motion
        if (form.style) params.style = form.style
        if (form.seed && form.seed > 0) params.seed = form.seed

        console.log('Generating video with params:', params)
        await videoAPI.generateVideo(params)
      }
      
      ElMessage.success('Video generation task submitted, please check back shortly')
      emit('success')
      handleClose()
    } catch (error: any) {
      console.error('Video generation failed:', error)
      ElMessage.error(error.response?.data?.message || error.message || 'Generation failed')
    } finally {
      generating.value = false
    }
  } catch (error: any) {
    console.error('Form validation error:', error)
    ElMessage.warning('Please check that all required fields are filled in')
  }
}

const handleClose = () => {
  visible.value = false
  formRef.value?.resetFields()
}
</script>

<style scoped>
.form-tip {
  margin-top: 4px;
  font-size: 12px;
  color: #999;
}

.slider-value {
  margin-left: 12px;
  font-size: 14px;
  font-weight: 500;
  color: #409eff;
}

.image-option {
  display: flex;
  align-items: center;
  gap: 8px;
}

.image-thumb {
  width: 40px;
  height: 40px;
  object-fit: cover;
  border-radius: 4px;
}
</style>
