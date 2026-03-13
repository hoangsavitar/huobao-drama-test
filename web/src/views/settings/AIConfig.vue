<template>
  <div class="page-container">
    <div class="content-wrapper animate-fade-in">
      <PageHeader
        :title="$t('aiConfig.title')"
        :subtitle="$t('aiConfig.subtitle') || 'Manage AI service configurations'"
        :show-back="true"
        :back-text="$t('common.back')"
      >
        <template #actions>
          <el-button type="primary" @click="showCreateDialog">
            <el-icon><Plus /></el-icon>
            <span>{{ $t("aiConfig.addConfig") }}</span>
          </el-button>
        </template>
      </PageHeader>

      <div class="tabs-wrapper">
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
      </div>

      <!-- Edit/Create Dialog -->
      <el-dialog
        v-model="dialogVisible"
        :title="isEdit ? $t('aiConfig.editConfig') : $t('aiConfig.addConfig')"
        width="600px"
        :close-on-click-modal="false"
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
          <el-button @click="dialogVisible = false">{{
            $t("common.cancel")
          }}</el-button>
          <el-button
            v-if="form.service_type === 'text'"
            @click="testConnection"
            :loading="testing"
            >{{ $t("aiConfig.actions.test") }}</el-button
          >
          <el-button type="primary" @click="handleSubmit" :loading="submitting">
            {{ isEdit ? $t("common.save") : $t("common.create") }}
          </el-button>
        </template>
      </el-dialog>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, computed } from "vue";
import { useRouter } from "vue-router";
import {
  ElMessage,
  ElMessageBox,
  type FormInstance,
  type FormRules,
} from "element-plus";
import { Plus, ArrowLeft } from "@element-plus/icons-vue";
import { aiAPI } from "@/api/ai";
import { PageHeader } from "@/components/common";
import type {
  AIServiceConfig,
  AIServiceType,
  CreateAIConfigRequest,
  UpdateAIConfigRequest,
} from "@/types/ai";
import ConfigList from "./components/ConfigList.vue";

const router = useRouter();

const activeTab = ref<AIServiceType>("text");
const loading = ref(false);
const configs = ref<AIServiceConfig[]>([]);
const dialogVisible = ref(false);
const isEdit = ref(false);
const editingId = ref<number>();
const formRef = ref<FormInstance>();
const submitting = ref(false);
const testing = ref(false);

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
      models: ["gpt-5.2", "gemini-3-flash-preview"],
    },
    {
      id: "chatfire",
      name: "Chatfire",
      models: [
        "gemini-3-flash-preview",
        "claude-sonnet-4-5-20250929",
        "doubao-seed-1-8-251228",
      ],
    },
    {
      id: "gemini",
      name: "Google Gemini",
      models: ["gemini-2.5-pro", "gemini-3-flash-preview"],
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
      models: ["doubao-seedream-4-5-251128", "nano-banana-pro"],
    },
    {
      id: "gemini",
      name: "Google Gemini",
      models: ["gemini-3-pro-image-preview"],
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
    { id: "openai", name: "OpenAI", models: ["sora-2", "sora-2-pro"] },
    //    { id: 'minimax', name: 'MiniMax', models: ['MiniMax-Hailuo-2.3', 'MiniMax-Hailuo-2.3-Fast', 'MiniMax-Hailuo-02'] }
  ],
};

const availableProviders = computed(() => {
  const activeConfigs = configs.value.filter(
    (c) => c.service_type === form.service_type && c.is_active,
  );
  const activeProviderIds = new Set(activeConfigs.map((c) => c.provider));
  const allProviders = providerConfigs[form.service_type] || [];
  return allProviders.filter((p) => activeProviderIds.has(p.id));
});

const availableModels = computed(() => {
  if (!form.provider) return [];

  const activeConfigsForProvider = configs.value.filter(
    (c) =>
      c.provider === form.provider &&
      c.service_type === form.service_type &&
      c.is_active,
  );

  const models = new Set<string>();
  activeConfigsForProvider.forEach((config) => {
    config.model.forEach((m) => models.add(m));
  });

  return Array.from(models);
});

// Full endpoint example
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
    } else if (provider === "openai") {
      endpoint = "/videos";
    } else {
      endpoint = "/video/generations";
    }
  }

  return baseUrl + endpoint;
});

const rules: FormRules = {
  name: [{ required: true, message: "Please enter a config name", trigger: "blur" }],
  provider: [{ required: true, message: "Please select a provider", trigger: "change" }],
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
  dialogVisible.value = true;
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
  dialogVisible.value = true;
};

const handleDelete = async (config: AIServiceConfig) => {
  try {
    await ElMessageBox.confirm("Are you sure to delete this config?", "Warning", {
      confirmButtonText: "Confirm",
      cancelButtonText: "Cancel",
      type: "warning",
    });

    await aiAPI.delete(config.id);
    ElMessage.success("Deleted successfully");
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
    ElMessage.success("Connection test successful!");
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
    ElMessage.success("Connection test successful!");
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
        ElMessage.success("Updated successfully");
      } else {
        await aiAPI.create(form);
        ElMessage.success("Created successfully");
      }

      dialogVisible.value = false;
      loadConfigs();
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

  if (form.provider === "gemini" || form.provider === "google") {
    form.base_url = "https://api.chatfire.site";
  } else {
    form.base_url = "https://api.chatfire.site/v1";
  }

  if (!isEdit.value) {
    form.name = generateConfigName(form.provider, form.service_type);
  }
};

// Endpoint is set automatically by the backend based on provider
const getDefaultEndpoint = (serviceType: AIServiceType): string => {
  switch (serviceType) {
    case "text":
      return "";
    case "image":
      return "/v1/images/generations";
    case "video":
      return "/v1/video/generations";
    default:
      return "/v1/chat/completions";
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

const goBack = () => {
  router.back();
};

onMounted(() => {
  loadConfigs();
});
</script>

<style scoped>
/* ========================================
   Page Layout - Compact spacing
   ======================================== */
.page-container {
  min-height: 100vh;
  background: var(--bg-primary);
  padding: var(--space-2) var(--space-3);
  transition: background var(--transition-normal);
}

@media (min-width: 768px) {
  .page-container {
    padding: var(--space-3) var(--space-4);
  }
}

@media (min-width: 1024px) {
  .page-container {
    padding: var(--space-4) var(--space-5);
  }
}

.content-wrapper {
  max-width: 1200px;
  margin: 0 auto;
}

/* ========================================
   Tabs - Compact padding
   ======================================== */
.tabs-wrapper {
  background: var(--bg-card);
  border: 1px solid var(--border-primary);
  border-radius: var(--radius-lg);
  padding: var(--space-3);
  box-shadow: var(--shadow-card);
}

@media (min-width: 768px) {
  .tabs-wrapper {
    padding: var(--space-4);
  }
}

/* ========================================
   Form Tips
   ======================================== */
.form-tip {
  font-size: 0.75rem;
  color: var(--text-muted);
  margin-top: 0.25rem;
}

/* ========================================
   Dialog
   ======================================== */
:deep(.el-dialog) {
  border-radius: 0.75rem;
}

:deep(.el-dialog__header) {
  padding: 1.25rem 1.5rem;
  border-bottom: 1px solid var(--border-primary);
  margin-right: 0;
}

:deep(.el-dialog__title) {
  font-size: 1.125rem;
  font-weight: 600;
  color: var(--text-primary);
}

:deep(.el-dialog__body) {
  padding: 1.5rem;
}

:deep(.el-dialog__footer) {
  padding: 1rem 1.5rem;
  border-top: 1px solid var(--border-primary);
}

/* ========================================
   Dark Mode
   ======================================== */
.dark .tabs-wrapper {
  background: var(--bg-card);
}

.dark :deep(.el-dialog) {
  background: var(--bg-card);
}

.dark :deep(.el-dialog__header) {
  background: var(--bg-card);
}

.dark :deep(.el-dialog__body) {
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

.dark :deep(.el-input__inner::placeholder) {
  color: var(--text-muted);
}

.dark :deep(.el-select .el-input__wrapper) {
  background: var(--bg-secondary);
}

.dark :deep(.el-textarea__inner) {
  background: var(--bg-secondary);
  color: var(--text-primary);
  box-shadow: 0 0 0 1px var(--border-primary) inset;
}

.dark :deep(.el-input-number) {
  background: var(--bg-secondary);
}

.dark :deep(.el-switch__core) {
  background: var(--bg-secondary);
  border-color: var(--border-primary);
}

.dark :deep(.el-button--default) {
  background: var(--bg-secondary);
  border-color: var(--border-primary);
  color: var(--text-primary);
}

.dark :deep(.el-button--default:hover) {
  background: var(--bg-card-hover);
  border-color: var(--border-secondary);
}
</style>
