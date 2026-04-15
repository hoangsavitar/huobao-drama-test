<template>
  <el-dialog
    v-model="visible"
    :title="$t('aiConfig.title')"
    width="900px"
    :close-on-click-modal="false"
    destroy-on-close
    class="ai-config-dialog"
  >
    <!-- Dialog Header Actions -->
    <template #header>
      <div class="dialog-header">
        <span class="dialog-title">{{ $t("aiConfig.title") }}</span>
        <div class="header-actions">
          <el-button type="success" size="small" @click="showQuickSetupDialog">
            <el-icon><MagicStick /></el-icon>
            <span>Quick Setup</span>
          </el-button>
          <el-button type="primary" size="small" @click="showCreateDialog">
            <el-icon><Plus /></el-icon>
            <span>{{ $t("aiConfig.addConfig") }}</span>
          </el-button>
        </div>
      </div>
    </template>

    <!-- Tabs -->
    <el-tabs
      v-model="activeTab"
      @tab-change="handleTabChange"
      class="config-tabs"
    >
      <el-tab-pane :label="$t('aiConfig.tabs.text')" name="text">
        <ConfigList
          :configs="configs"
          :loading="loading"
          :show-test-button="true"
          @edit="handleEdit"
          @delete="handleDelete"
          @toggle-active="handleToggleActive"
          @test="handleTest"
        />
      </el-tab-pane>

      <el-tab-pane :label="$t('aiConfig.tabs.image')" name="image">
        <ConfigList
          :configs="configs"
          :loading="loading"
          :show-test-button="false"
          @edit="handleEdit"
          @delete="handleDelete"
          @toggle-active="handleToggleActive"
        />
      </el-tab-pane>

      <el-tab-pane :label="$t('aiConfig.tabs.video')" name="video">
        <ConfigList
          :configs="configs"
          :loading="loading"
          :show-test-button="false"
          @edit="handleEdit"
          @delete="handleDelete"
          @toggle-active="handleToggleActive"
        />
      </el-tab-pane>
    </el-tabs>

    <!-- Quick Setup Dialog -->
    <el-dialog
      v-model="quickSetupVisible"
      title="Quick Setup"
      width="500px"
      :close-on-click-modal="false"
      append-to-body
    >
      <div class="quick-setup-info">
        <p>The following configs will be created automatically:</p>
        <ul>
          <li>
            <strong>Text Service</strong>: {{ providerConfigs.text[1].models[0] }}
          </li>
          <li>
            <strong>Image Service</strong>: {{ providerConfigs.image[1].models[0] }}
          </li>
          <li>
            <strong>Video Service</strong>: {{ providerConfigs.video[1].models[0] }}
          </li>
        </ul>
        <p class="quick-setup-tip">Base URL: https://api.chatfire.site/v1</p>
      </div>
      <el-form label-width="80px">
        <el-form-item label="API Key" required>
          <el-input
            v-model="quickSetupApiKey"
            type="password"
            show-password
            placeholder="Enter ChatFire API Key"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <div class="quick-setup-footer">
          <a
            href="https://api.chatfire.site/login?inviteCode=C4453345"
            target="_blank"
            class="register-link"
          >
            No API Key? Click to register
          </a>
          <div class="footer-buttons">
            <el-button @click="quickSetupVisible = false">Cancel</el-button>
            <el-button
              type="primary"
              @click="handleQuickSetup"
              :loading="quickSetupLoading"
            >
              Confirm Setup
            </el-button>
          </div>
        </div>
      </template>
    </el-dialog>

    <!-- Edit/Create Sub-Dialog -->
    <el-dialog
      v-model="editDialogVisible"
      :title="isEdit ? $t('aiConfig.editConfig') : $t('aiConfig.addConfig')"
      width="600px"
      :close-on-click-modal="false"
      append-to-body
    >
      <el-form ref="formRef" :model="form" :rules="rules" label-width="100px">
        <el-form-item :label="$t('aiConfig.form.name')" prop="name">
          <el-input
            v-model="form.name"
            :placeholder="$t('aiConfig.form.namePlaceholder')"
          />
        </el-form-item>

        <el-form-item :label="$t('aiConfig.form.provider')" prop="provider">
          <el-select
            v-model="form.provider"
            :placeholder="$t('aiConfig.form.providerPlaceholder')"
            @change="handleProviderChange"
            style="width: 100%"
          >
            <el-option
              v-for="provider in availableProviders"
              :key="provider.id"
              :label="provider.name"
              :value="provider.id"
              :disabled="provider.disabled"
            />
          </el-select>
          <div class="form-tip">{{ $t("aiConfig.form.providerTip") }}</div>
        </el-form-item>

        <el-form-item :label="$t('aiConfig.form.priority')" prop="priority">
          <el-input-number
            v-model="form.priority"
            :min="0"
            :max="100"
            :step="1"
            style="width: 100%"
          />
          <div class="form-tip">{{ $t("aiConfig.form.priorityTip") }}</div>
        </el-form-item>

        <el-form-item :label="$t('aiConfig.form.model')" prop="model">
          <el-select
            v-model="form.model"
            :placeholder="$t('aiConfig.form.modelPlaceholder')"
            multiple
            filterable
            allow-create
            default-first-option
            collapse-tags
            collapse-tags-tooltip
            style="width: 100%"
          >
            <el-option
              v-for="model in availableModels"
              :key="model"
              :label="model"
              :value="model"
            />
          </el-select>
          <div class="form-tip">{{ $t("aiConfig.form.modelTip") }}</div>
        </el-form-item>

        <el-form-item :label="$t('aiConfig.form.baseUrl')" prop="base_url">
          <el-input
            v-model="form.base_url"
            :placeholder="$t('aiConfig.form.baseUrlPlaceholder')"
          />
          <div class="form-tip">
            {{ $t("aiConfig.form.baseUrlTip") }}
            <br />
            {{ $t("aiConfig.form.fullEndpoint") }}: {{ fullEndpointExample }}
          </div>
        </el-form-item>

        <el-form-item :label="$t('aiConfig.form.apiKey')" prop="api_key">
          <el-input
            v-model="form.api_key"
            type="password"
            show-password
            :placeholder="$t('aiConfig.form.apiKeyPlaceholder')"
          />
          <div class="form-tip">{{ $t("aiConfig.form.apiKeyTip") }}</div>
        </el-form-item>

        <el-form-item v-if="isEdit" :label="$t('aiConfig.form.isActive')">
          <el-switch v-model="form.is_active" />
        </el-form-item>
      </el-form>

      <template #footer>
        <div class="quick-setup-footer">
          <a
            href="https://api.chatfire.site/login?inviteCode=C4453345"
            target="_blank"
            class="register-link"
          >
            No API Key? Click to register
          </a>
          <div class="footer-buttons">
            <el-button @click="editDialogVisible = false">{{
              $t("common.cancel")
            }}</el-button>
            <el-button
              v-if="form.service_type === 'text'"
              @click="testConnection"
              :loading="testing"
              >{{ $t("aiConfig.actions.test") }}</el-button
            >
            <el-button
              type="primary"
              @click="handleSubmit"
              :loading="submitting"
            >
              {{ isEdit ? $t("common.save") : $t("common.create") }}
            </el-button>
          </div>
        </div>
      </template>
    </el-dialog>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, reactive, computed, watch } from "vue";
import {
  ElMessage,
  ElMessageBox,
  type FormInstance,
  type FormRules,
} from "element-plus";
import { Plus, MagicStick } from "@element-plus/icons-vue";
import { aiAPI } from "@/api/ai";
import type {
  AIServiceConfig,
  AIServiceType,
  CreateAIConfigRequest,
  UpdateAIConfigRequest,
} from "@/types/ai";
import ConfigList from "@/views/settings/components/ConfigList.vue";

const props = defineProps<{
  modelValue: boolean;
}>();

const emit = defineEmits<{
  "update:modelValue": [value: boolean];
  "config-updated": [];
}>();

const visible = computed({
  get: () => props.modelValue,
  set: (val) => emit("update:modelValue", val),
});

const activeTab = ref<AIServiceType>("text");
const loading = ref(false);
const configs = ref<AIServiceConfig[]>([]);
const editDialogVisible = ref(false);
const isEdit = ref(false);
const editingId = ref<number>();
const formRef = ref<FormInstance>();
const submitting = ref(false);
const testing = ref(false);
const quickSetupVisible = ref(false);
const quickSetupApiKey = ref("");
const quickSetupLoading = ref(false);

const form = reactive<
  CreateAIConfigRequest & { is_active?: boolean; provider?: string }
>({
  service_type: "text",
  provider: "",
  name: "",
  base_url: "",
  api_key: "",
  model: [],
  priority: 0,
  is_active: true,
});

// Provider configs
interface ProviderConfig {
  id: string;
  name: string;
  models: string[];
  disabled?: boolean;
}

const providerConfigs: Record<AIServiceType, ProviderConfig[]> = {
  text: [
    {
      id: "openai",
      name: "OpenAI",
      models: ["gpt-5.2", "gemini-3.1-flash-lite-preview", "gemini-3-flash-preview"],
    },
    {
      id: "chatfire",
      name: "Chatfire",
      models: [
        "gemini-3.1-flash-lite-preview",
        "gemini-3-flash-preview",
        "claude-sonnet-4-5-20250929",
        "doubao-seed-1-8-251228",
      ],
    },
    {
      id: "gemini",
      name: "Google Gemini",
      models: ["gemini-3.1-flash-lite-preview", "gemini-2.5-pro", "gemini-3-flash-preview"],
    },
  ],
  image: [
    {
      id: "volcengine",
      name: "Volcengine",
      models: ["doubao-seedream-4-5-251128", "doubao-seedream-4-0-250828"],
    },
    {
      id: "chatfire",
      name: "Chatfire",
      models: ["nano-banana-pro", "doubao-seedream-4-5-251128"],
    },
    {
      id: "gemini",
      name: "Google Gemini",
      models: ["gemini-3.1-flash-image-preview", "gemini-3-pro-image-preview"],
    },
    { id: "openai", name: "OpenAI", models: ["dall-e-3", "dall-e-2"] },
  ],
  video: [
    {
      id: "volces",
      name: "Volcengine",
      models: [
        "doubao-seedance-1-5-pro-251215",
        "doubao-seedance-1-0-lite-i2v-250428",
        "doubao-seedance-1-0-lite-t2v-250428",
        "doubao-seedance-1-0-pro-250528",
        "doubao-seedance-1-0-pro-fast-251015",
      ],
    },
    {
      id: "chatfire",
      name: "Chatfire",
      models: [
        "doubao-seedance-1-5-pro-251215",
        "doubao-seedance-1-0-lite-i2v-250428",
        "doubao-seedance-1-0-lite-t2v-250428",
        "doubao-seedance-1-0-pro-250528",
        "doubao-seedance-1-0-pro-fast-251015",
        "sora-2",
        "sora-2-pro",
      ],
    },
    {
      id: "minimax",
      name: "MiniMax Hailuo",
      models: [
        "MiniMax-Hailuo-2.3",
        "MiniMax-Hailuo-2.3-Fast",
        "MiniMax-Hailuo-02",
      ],
    },
    { id: "openai", name: "OpenAI", models: ["sora-2", "sora-2-pro"] },
  ],
};

// 当前可用的厂商列表（显示所有配置的厂商）
const availableProviders = computed(() => {
  // 返回当前service_type下的所有厂商
  return providerConfigs[form.service_type] || [];
});

// 当前可用的模型列表（从预定义配置中获取）
const availableModels = computed(() => {
  if (!form.provider || !form.service_type) return [];

  // 从预定义配置中查找当前厂商的模型列表
  const providerConfig = providerConfigs[form.service_type]?.find(
    (p) => p.id === form.provider,
  );

  return providerConfig?.models || [];
});

const fullEndpointExample = computed(() => {
  const baseUrl = form.base_url || "https://api.example.com";
  const provider = form.provider;
  const serviceType = form.service_type;

  let endpoint = "";

  if (serviceType === "text") {
    if (provider === "gemini" || provider === "google") {
      endpoint = "/v1beta/models/{model}:generateContent";
    } else {
      endpoint = "/chat/completions";
    }
  } else if (serviceType === "image") {
    if (provider === "gemini" || provider === "google") {
      endpoint = "/v1beta/models/{model}:generateContent";
    } else {
      endpoint = "/images/generations";
    }
  } else if (serviceType === "video") {
    if (provider === "chatfire") {
      endpoint = "/video/generations";
    } else if (
      provider === "doubao" ||
      provider === "volcengine" ||
      provider === "volces"
    ) {
      endpoint = "/contents/generations/tasks";
    } else if (provider === "minimax") {
      endpoint = "/video_generation";
    } else if (provider === "openai") {
      endpoint = "/videos";
    } else {
      endpoint = "/video/generations";
    }
  }

  return baseUrl + endpoint;
});

const rules: FormRules = {
  name: [{ required: true, message: "Please enter config name", trigger: "blur" }],
  provider: [{ required: true, message: "Please select provider", trigger: "change" }],
  base_url: [
    { required: true, message: "Please enter Base URL", trigger: "blur" },
    { type: "url", message: "Please enter a valid URL", trigger: "blur" },
  ],
  api_key: [{ required: true, message: "Please enter API Key", trigger: "blur" }],
  model: [
    {
      required: true,
      message: "Please select at least one model",
      trigger: "change",
      validator: (rule: any, value: any, callback: any) => {
        if (Array.isArray(value) && value.length > 0) {
          callback();
        } else if (typeof value === "string" && value.length > 0) {
          callback();
        } else {
          callback(new Error("Please select at least one model"));
        }
      },
    },
  ],
};

const loadConfigs = async () => {
  loading.value = true;
  try {
    configs.value = await aiAPI.list(activeTab.value);
  } catch (error: any) {
    ElMessage.error(error.message || "Load failed");
  } finally {
    loading.value = false;
  }
};

const generateConfigName = (
  provider: string,
  serviceType: AIServiceType,
): string => {
  const providerNames: Record<string, string> = {
    chatfire: "ChatFire",
    openai: "OpenAI",
    gemini: "Gemini",
    google: "Google",
  };

  const serviceNames: Record<AIServiceType, string> = {
    text: "Text",
    image: "Image",
    video: "Video",
  };

  const randomNum = Math.floor(Math.random() * 10000)
    .toString()
    .padStart(4, "0");
  const providerName = providerNames[provider] || provider;
  const serviceName = serviceNames[serviceType] || serviceType;

  return `${providerName}-${serviceName}-${randomNum}`;
};

const showCreateDialog = () => {
  isEdit.value = false;
  editingId.value = undefined;
  resetForm();
  form.service_type = activeTab.value;
  form.provider = "chatfire";
  form.base_url = "https://api.chatfire.site/v1";
  form.name = generateConfigName("chatfire", activeTab.value);
  // 图片模型配置默认 nano
  if (activeTab.value === "image") {
    form.model = ["nano-banana-pro"];
  }
  editDialogVisible.value = true;
};

const handleEdit = (config: AIServiceConfig) => {
  isEdit.value = true;
  editingId.value = config.id;

  Object.assign(form, {
    service_type: config.service_type,
    provider: config.provider || "chatfire",
    name: config.name,
    base_url: config.base_url,
    api_key: config.api_key,
    model: Array.isArray(config.model) ? config.model : [config.model],
    priority: config.priority || 0,
    is_active: config.is_active,
  });
  editDialogVisible.value = true;
};

const handleDelete = async (config: AIServiceConfig) => {
  try {
    await ElMessageBox.confirm("Delete this config?", "Warning", {
      confirmButtonText: "Delete",
      cancelButtonText: "Cancel",
      type: "warning",
    });

    await aiAPI.delete(config.id);
    ElMessage.success("Deleted");
    loadConfigs();
  } catch (error: any) {
    if (error !== "cancel") {
      ElMessage.error(error.message || "Delete failed");
    }
  }
};

const handleToggleActive = async (config: AIServiceConfig) => {
  try {
    const newActiveState = !config.is_active;
    await aiAPI.update(config.id, { is_active: newActiveState });
    ElMessage.success(newActiveState ? "Config enabled" : "Config disabled");
    await loadConfigs();
  } catch (error: any) {
    ElMessage.error(error.message || "Operation failed");
  }
};

const testConnection = async () => {
  if (!formRef.value) return;

  const valid = await formRef.value.validate().catch(() => false);
  if (!valid) return;

  testing.value = true;
  try {
    await aiAPI.testConnection({
      base_url: form.base_url,
      api_key: form.api_key,
      model: form.model,
      provider: form.provider,
    });
    ElMessage.success("Connection test successful");
  } catch (error: any) {
    ElMessage.error(error.message || "Connection test failed");
  } finally {
    testing.value = false;
  }
};

const handleTest = async (config: AIServiceConfig) => {
  testing.value = true;
  try {
    await aiAPI.testConnection({
      base_url: config.base_url,
      api_key: config.api_key,
      model: config.model,
      provider: config.provider,
    });
    ElMessage.success("Connection test successful");
  } catch (error: any) {
    ElMessage.error(error.message || "Connection test failed");
  } finally {
    testing.value = false;
  }
};

const handleSubmit = async () => {
  if (!formRef.value) return;

  await formRef.value.validate(async (valid) => {
    if (!valid) return;

    submitting.value = true;
    try {
      if (isEdit.value && editingId.value) {
        const updateData: UpdateAIConfigRequest = {
          name: form.name,
          provider: form.provider,
          base_url: form.base_url,
          api_key: form.api_key,
          model: form.model,
          priority: form.priority,
          is_active: form.is_active,
        };
        await aiAPI.update(editingId.value, updateData);
        ElMessage.success("Updated");
      } else {
        await aiAPI.create(form);
        ElMessage.success("Created");
      }

      editDialogVisible.value = false;
      loadConfigs();
      emit("config-updated");
    } catch (error: any) {
      ElMessage.error(error.message || "Operation failed");
    } finally {
      submitting.value = false;
    }
  });
};

const handleTabChange = (tabName: string | number) => {
  activeTab.value = tabName as AIServiceType;
  loadConfigs();
};

const handleProviderChange = () => {
  form.model = [];

  // 根据厂商自动设置 Base URL
  if (form.provider === "gemini" || form.provider === "google") {
    form.base_url = "https://generativelanguage.googleapis.com";
  } else if (form.provider === "minimax") {
    form.base_url = "https://api.minimaxi.com/v1";
  } else if (form.provider === "volces" || form.provider === "volcengine") {
    form.base_url = "https://ark.cn-beijing.volces.com/api/v3";
  } else if (form.provider === "openai") {
    form.base_url = "https://api.openai.com/v1";
  } else {
    // chatfire 和其他厂商
    form.base_url = "https://api.chatfire.site/v1";
  }

  if (!isEdit.value) {
    form.name = generateConfigName(form.provider, form.service_type);
  }
};

const resetForm = () => {
  const serviceType = form.service_type || "text";
  Object.assign(form, {
    service_type: serviceType,
    provider: "",
    name: "",
    base_url: "",
    api_key: "",
    model: [],
    priority: 0,
    is_active: true,
  });
  formRef.value?.resetFields();
};

const showQuickSetupDialog = () => {
  quickSetupApiKey.value = "";
  quickSetupVisible.value = true;
};

const handleQuickSetup = async () => {
  if (!quickSetupApiKey.value.trim()) {
    ElMessage.warning("Please enter API Key");
    return;
  }

  quickSetupLoading.value = true;
  const baseUrl = "https://api.chatfire.site/v1";
  const apiKey = quickSetupApiKey.value.trim();

  try {
    // 加载所有类型的配置，检查是否已存在相同 baseUrl 的配置
    const [textConfigs, imageConfigs, videoConfigs] = await Promise.all([
      aiAPI.list("text"),
      aiAPI.list("image"),
      aiAPI.list("video"),
    ]);

    const createdServices: string[] = [];
    const skippedServices: string[] = [];

    // 创建文本配置（如果不存在）
    const existingTextConfig = textConfigs.find((c) => c.base_url === baseUrl);
    if (!existingTextConfig) {
      const textProvider = providerConfigs.text.find(
        (p) => p.id === "chatfire",
      )!;
      await aiAPI.create({
        service_type: "text",
        provider: "chatfire",
        name: generateConfigName("chatfire", "text"),
        base_url: baseUrl,
        api_key: apiKey,
        model: [textProvider.models[0]],
        priority: 0,
      });
      createdServices.push("Text");
    } else {
      skippedServices.push("Text");
    }

    // 创建图片配置（如果不存在）
    const existingImageConfig = imageConfigs.find(
      (c) => c.base_url === baseUrl,
    );
    if (!existingImageConfig) {
      const imageProvider = providerConfigs.image.find(
        (p) => p.id === "chatfire",
      )!;
      await aiAPI.create({
        service_type: "image",
        provider: "chatfire",
        name: generateConfigName("chatfire", "image"),
        base_url: baseUrl,
        api_key: apiKey,
        model: [imageProvider.models[0]],
        priority: 0,
      });
      createdServices.push("Image");
    } else {
      skippedServices.push("Image");
    }

    // 创建视频配置（如果不存在）
    const existingVideoConfig = videoConfigs.find(
      (c) => c.base_url === baseUrl,
    );
    if (!existingVideoConfig) {
      const videoProvider = providerConfigs.video.find(
        (p) => p.id === "chatfire",
      )!;
      await aiAPI.create({
        service_type: "video",
        provider: "chatfire",
        name: generateConfigName("chatfire", "video"),
        base_url: baseUrl,
        api_key: apiKey,
        model: [videoProvider.models[0]],
        priority: 0,
      });
      createdServices.push("Video");
    } else {
      skippedServices.push("Video");
    }

    // 显示结果消息
    if (createdServices.length > 0 && skippedServices.length > 0) {
      ElMessage.success(
        `Created ${createdServices.join(", ")} configs; ${skippedServices.join(", ")} already exist`,
      );
    } else if (createdServices.length > 0) {
      ElMessage.success(
        `Quick setup completed. Created ${createdServices.join(", ")} service configs`,
      );
    } else {
      ElMessage.info("All configs already exist");
    }

    quickSetupVisible.value = false;
    loadConfigs();
    if (createdServices.length > 0) {
      emit("config-updated");
    }
  } catch (error: any) {
    ElMessage.error(error.message || "Configuration failed");
  } finally {
    quickSetupLoading.value = false;
  }
};

// Load configs when dialog opens
watch(visible, (val) => {
  if (val) {
    loadConfigs();
  }
});
</script>

<style scoped>
.ai-config-dialog :deep(.el-dialog__header) {
  padding: 16px 20px;
  border-bottom: 1px solid var(--border-primary);
  margin-right: 0;
}

.dialog-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  padding-right: 32px;
}

.header-actions {
  display: flex;
  gap: 8px;
}

.quick-setup-info {
  margin-bottom: 16px;
  padding: 12px 16px;
  background: var(--bg-secondary);
  border-radius: 8px;
  font-size: 14px;
  color: var(--text-primary);

  p {
    margin: 0 0 8px 0;
  }

  ul {
    margin: 8px 0;
    padding-left: 20px;
  }

  li {
    margin: 4px 0;
    color: var(--text-secondary);
  }

  .quick-setup-tip {
    margin-top: 12px;
    font-size: 12px;
    color: var(--text-muted);
  }
}

.quick-setup-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  width: 100%;
}

.register-link {
  font-size: 12px;
  color: var(--text-muted);
  text-decoration: none;
  transition: color 0.2s;

  &:hover {
    color: var(--accent);
  }
}

.footer-buttons {
  display: flex;
  gap: 8px;
}

.dialog-title {
  font-size: 1.125rem;
  font-weight: 600;
  color: var(--text-primary);
}

.ai-config-dialog :deep(.el-dialog__body) {
  padding: 20px;
  max-height: 60vh;
  overflow-y: auto;
}

.config-tabs {
  margin: 0;
}

.form-tip {
  font-size: 0.75rem;
  color: var(--text-muted);
  margin-top: 0.25rem;
  word-break: break-all;
  overflow-wrap: break-word;
  line-height: 1.5;
}

/* Dark mode */
.dark .ai-config-dialog :deep(.el-dialog) {
  background: var(--bg-card);
}

.dark .ai-config-dialog :deep(.el-dialog__header) {
  background: var(--bg-card);
}

.dark :deep(.el-form-item__label) {
  color: var(--text-primary);
}

.dark :deep(.el-input__wrapper) {
  background: var(--bg-secondary);
  box-shadow: 0 0 0 1px var(--border-primary) inset;
}

.dark :deep(.el-input__inner) {
  color: var(--text-primary);
}

.dark :deep(.el-select .el-input__wrapper) {
  background: var(--bg-secondary);
}
</style>
