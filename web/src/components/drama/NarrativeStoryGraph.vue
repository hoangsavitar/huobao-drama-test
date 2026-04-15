<template>
  <el-card shadow="never" class="narrative-graph-card">
    <template #header>
      <span>{{ title }}</span>
    </template>
    <div v-if="!source?.trim()" class="graph-empty">
      {{ emptyText }}
    </div>
    <div v-else class="mermaid-wrap">
      <div ref="hostRef" class="mermaid-host" />
    </div>
  </el-card>
</template>

<script setup lang="ts">
import { nextTick, onBeforeUnmount, onMounted, ref, watch } from "vue";
import mermaid from "mermaid";

const props = withDefaults(
  defineProps<{
    source: string;
    title?: string;
    emptyText?: string;
  }>(),
  {
    title: "Story graph",
    emptyText: "No graph data",
  },
);

const hostRef = ref<HTMLElement | null>(null);
let renderId = 0;

mermaid.initialize({
  startOnLoad: false,
  securityLevel: "loose",
  theme: "neutral",
});

async function render() {
  const host = hostRef.value;
  if (!host || !props.source?.trim()) return;
  const id = ++renderId;
  host.innerHTML = "";
  const graphId = `ng-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
  try {
    const { svg } = await mermaid.render(graphId, props.source);
    if (id !== renderId) return;
    host.innerHTML = svg;
  } catch (e) {
    console.warn("Mermaid render failed", e);
    if (id !== renderId) return;
    host.innerHTML = `<pre class="mermaid-fallback">${escapeHtml(props.source)}</pre>`;
  }
}

function escapeHtml(s: string) {
  return s
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;");
}

watch(
  () => props.source,
  () => {
    void nextTick().then(() => render());
  },
);

onMounted(() => {
  void nextTick().then(() => render());
});

onBeforeUnmount(() => {
  renderId++;
});
</script>

<style scoped>
.narrative-graph-card {
  margin-bottom: 16px;
}
.mermaid-wrap {
  overflow: auto;
  max-height: 420px;
  background: var(--el-fill-color-lighter);
  border-radius: 8px;
  padding: 12px;
}
.graph-empty {
  color: var(--el-text-color-secondary);
  padding: 16px;
}
.mermaid-fallback {
  font-size: 11px;
  white-space: pre-wrap;
  word-break: break-all;
}
</style>
