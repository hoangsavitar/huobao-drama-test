<template>
  <div class="story-graph-container">
    <!-- Header bar -->
    <div class="graph-header">
      <div class="graph-title-group">
        <span class="graph-icon">⬡</span>
        <span class="graph-title">{{ title }}</span>
        <span class="graph-badge">STORY GRAPH</span>
      </div>
      <div class="graph-header-right">
        <div class="graph-legend">
          <span class="legend-item entry">● Entry</span>
          <span class="legend-item node">● Node</span>
          <span class="legend-item branch">⬡ Branch</span>
        </div>
        <!-- Zoom controls (only show when graph has content) -->
        <div v-if="source?.trim()" class="zoom-controls">
          <button class="zoom-btn" title="Zoom out" @click="zoomOut">−</button>
          <span class="zoom-pct">{{ Math.round(scale * 100) }}%</span>
          <button class="zoom-btn" title="Zoom in" @click="zoomIn">+</button>
          <button class="zoom-btn zoom-reset" title="Reset view" @click="resetView">↺</button>
        </div>
      </div>
    </div>

    <!-- Empty state -->
    <div v-if="!source?.trim()" class="graph-empty">
      <div class="graph-empty-icon">⬡</div>
      <p>{{ emptyText }}</p>
      <p class="graph-empty-hint">Run Agent 1 to generate the story graph</p>
    </div>

    <!-- Graph render area (zoomable + pannable) -->
    <div
      v-else
      ref="viewportRef"
      class="mermaid-viewport"
      @wheel.prevent="onWheel"
      @mousedown="onMouseDown"
      @mousemove="onMouseMove"
      @mouseup="onMouseUp"
      @mouseleave="onMouseUp"
    >
      <div
        class="mermaid-canvas"
        :style="{
          transform: `translate(${panX}px, ${panY}px) scale(${scale})`,
          transformOrigin: '0 0',
          cursor: isDragging ? 'grabbing' : 'grab',
        }"
      >
        <div ref="hostRef" class="mermaid-host" />
      </div>
    </div>
  </div>
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
    title: "Narrative Graph",
    emptyText: "No graph data available",
  },
);

const hostRef = ref<HTMLElement | null>(null);
const viewportRef = ref<HTMLElement | null>(null);
let renderId = 0;

// ── Zoom & Pan state ──────────────────────────────────────────────
const scale = ref(1);
const panX = ref(0);
const panY = ref(0);
const isDragging = ref(false);
let dragStartX = 0;
let dragStartY = 0;
let dragStartPanX = 0;
let dragStartPanY = 0;

const MIN_SCALE = 0.2;
const MAX_SCALE = 4;
const ZOOM_STEP = 0.15;

function clampScale(s: number) {
  return Math.min(MAX_SCALE, Math.max(MIN_SCALE, s));
}

function zoomIn() {
  scale.value = clampScale(scale.value + ZOOM_STEP);
}

function zoomOut() {
  scale.value = clampScale(scale.value - ZOOM_STEP);
}

function fitToViewport() {
  const viewport = viewportRef.value;
  const host = hostRef.value;
  if (!viewport || !host) return;

  const svgEl = host.querySelector("svg");
  if (!svgEl) return;

  const viewBoxAttr = svgEl.getAttribute("viewBox");
  const widthAttr = svgEl.getAttribute("width");
  const heightAttr = svgEl.getAttribute("height");

  let svgWidth = 0;
  let svgHeight = 0;

  if (viewBoxAttr) {
    const parts = viewBoxAttr.split(" ").map(Number);
    if (parts.length === 4) {
      svgWidth = parts[2];
      svgHeight = parts[3];
    }
  }

  if (!svgWidth || !svgHeight) {
    if (widthAttr) svgWidth = parseFloat(widthAttr);
    if (heightAttr) svgHeight = parseFloat(heightAttr);
  }

  if (!svgWidth || !svgHeight) {
    const rect = svgEl.getBoundingClientRect();
    svgWidth = rect.width;
    svgHeight = rect.height;
  }

  if (!svgWidth || !svgHeight) return;

  const viewWidth = viewport.clientWidth;
  const viewHeight = viewport.clientHeight;

  const padding = 32;
  const scaleX = (viewWidth - padding * 2) / svgWidth;
  const scaleY = (viewHeight - padding * 2) / svgHeight;

  let fitScale = Math.min(scaleX, scaleY);
  // Restrict bounds so it fits nicely
  fitScale = Math.max(0.4, Math.min(1.8, fitScale));

  scale.value = fitScale;
  panX.value = (viewWidth - svgWidth * fitScale) / 2;
  panY.value = (viewHeight - svgHeight * fitScale) / 2;
}

function resetView() {
  void nextTick().then(() => {
    fitToViewport();
  });
}

function onWheel(e: WheelEvent) {
  const viewport = viewportRef.value;
  if (!viewport) return;

  const rect = viewport.getBoundingClientRect();
  // Mouse position relative to viewport
  const mouseX = e.clientX - rect.left;
  const mouseY = e.clientY - rect.top;

  const oldScale = scale.value;
  const delta = e.deltaY < 0 ? ZOOM_STEP : -ZOOM_STEP;
  const newScale = clampScale(oldScale + delta);

  // Zoom toward cursor
  const ratio = newScale / oldScale;
  panX.value = mouseX - ratio * (mouseX - panX.value);
  panY.value = mouseY - ratio * (mouseY - panY.value);
  scale.value = newScale;
}

function onMouseDown(e: MouseEvent) {
  if (e.button !== 0) return;
  isDragging.value = true;
  dragStartX = e.clientX;
  dragStartY = e.clientY;
  dragStartPanX = panX.value;
  dragStartPanY = panY.value;
}

function onMouseMove(e: MouseEvent) {
  if (!isDragging.value) return;
  panX.value = dragStartPanX + (e.clientX - dragStartX);
  panY.value = dragStartPanY + (e.clientY - dragStartY);
}

function onMouseUp() {
  isDragging.value = false;
}

// ── Mermaid rendering ─────────────────────────────────────────────
mermaid.initialize({
  startOnLoad: false,
  securityLevel: "loose",
  theme: "dark",
  darkMode: true,
  themeVariables: {
    background: "#0f1117",
    primaryColor: "#1e2235",
    primaryTextColor: "#e2e8f0",
    primaryBorderColor: "#4f46e5",
    lineColor: "#6366f1",
    secondaryColor: "#1a1f2e",
    tertiaryColor: "#131722",
    nodeBorder: "#4f46e5",
    clusterBkg: "#1a1f2e",
    titleColor: "#a5b4fc",
    edgeLabelBackground: "#0f1117",
    fontFamily: "Inter, system-ui, sans-serif",
    fontSize: "13px",
  },
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

    const svgEl = host.querySelector("svg");
    if (svgEl) {
      // Set width and height explicitly based on viewBox to prevent collapsing in inline-block wrapper
      const viewBoxAttr = svgEl.getAttribute("viewBox");
      if (viewBoxAttr) {
        const parts = viewBoxAttr.split(" ").map(Number);
        if (parts.length === 4) {
          const naturalWidth = parts[2];
          const naturalHeight = parts[3];
          if (naturalWidth && naturalHeight) {
            svgEl.setAttribute("width", naturalWidth.toString());
            svgEl.setAttribute("height", naturalHeight.toString());
          }
        }
      }
      svgEl.style.maxWidth = "none";
      svgEl.style.display = "block";
      svgEl.style.borderRadius = "6px";
    }
    // Reset zoom when new graph is rendered
    resetView();
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
.story-graph-container {
  background: #0b0e18;
  border: 1px solid rgba(99, 102, 241, 0.25);
  border-radius: 12px;
  overflow: hidden;
  margin-bottom: 16px;
  box-shadow: 0 4px 32px rgba(0, 0, 0, 0.4), 0 0 0 1px rgba(99, 102, 241, 0.1);
  user-select: none;
}

.graph-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 16px;
  background: linear-gradient(90deg, rgba(99, 102, 241, 0.12) 0%, rgba(15, 17, 23, 0) 100%);
  border-bottom: 1px solid rgba(99, 102, 241, 0.18);
}

.graph-header-right {
  display: flex;
  align-items: center;
  gap: 16px;
}

.graph-title-group {
  display: flex;
  align-items: center;
  gap: 10px;
}

.graph-icon {
  font-size: 16px;
  color: #6366f1;
  line-height: 1;
}

.graph-title {
  font-size: 13px;
  font-weight: 700;
  color: #c7d2fe;
  letter-spacing: 0.3px;
}

.graph-badge {
  font-size: 9px;
  font-weight: 800;
  letter-spacing: 1.2px;
  color: #6366f1;
  background: rgba(99, 102, 241, 0.12);
  border: 1px solid rgba(99, 102, 241, 0.3);
  border-radius: 4px;
  padding: 2px 7px;
  text-transform: uppercase;
}

.graph-legend {
  display: flex;
  align-items: center;
  gap: 14px;
}

.legend-item {
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 0.3px;
}

.legend-item.entry  { color: #34d399; }
.legend-item.node   { color: #a5b4fc; }
.legend-item.branch { color: #fb923c; }

/* ── Zoom controls ── */
.zoom-controls {
  display: flex;
  align-items: center;
  gap: 4px;
  background: rgba(15, 17, 23, 0.7);
  border: 1px solid rgba(99, 102, 241, 0.25);
  border-radius: 8px;
  padding: 3px 6px;
}

.zoom-btn {
  width: 24px;
  height: 24px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: transparent;
  border: none;
  border-radius: 5px;
  color: #a5b4fc;
  font-size: 16px;
  font-weight: 700;
  cursor: pointer;
  line-height: 1;
  padding: 0;
  transition: background 0.15s, color 0.15s;
}

.zoom-btn:hover {
  background: rgba(99, 102, 241, 0.2);
  color: #e0e7ff;
}

.zoom-btn.zoom-reset {
  font-size: 14px;
  color: #64748b;
}

.zoom-btn.zoom-reset:hover {
  color: #94a3b8;
}

.zoom-pct {
  font-size: 11px;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
  color: #64748b;
  min-width: 36px;
  text-align: center;
}

/* ── Viewport (clipping container) ── */
.mermaid-viewport {
  height: 480px;
  overflow: hidden;
  background: #0b0e18;
  position: relative;
  cursor: grab;
}

.mermaid-viewport:active {
  cursor: grabbing;
}

/* ── Canvas (transformed layer) ── */
.mermaid-canvas {
  display: inline-block;
  padding: 24px;
}

.mermaid-host {
  min-height: 80px;
}

/* ── Empty state ── */
.graph-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 48px 24px;
  text-align: center;
  color: #475569;
}

.graph-empty-icon {
  font-size: 40px;
  margin-bottom: 12px;
  opacity: 0.3;
}

.graph-empty p {
  margin: 0 0 6px;
  font-size: 14px;
  color: #64748b;
}

.graph-empty-hint {
  font-size: 12px !important;
  color: #334155 !important;
  font-style: italic;
}

/* ── Mermaid SVG overrides ── */
.mermaid-host :deep(svg) {
  background: transparent !important;
  image-rendering: -webkit-optimize-contrast;
  image-rendering: crisp-edges;
}

.mermaid-host :deep(.node rect),
.mermaid-host :deep(.node circle),
.mermaid-host :deep(.node polygon) {
  stroke: #4f46e5 !important;
}

.mermaid-host :deep(.edgePath path) {
  stroke: #6366f1 !important;
}

.mermaid-host :deep(.edgeLabel) {
  background: #0f1117 !important;
  color: #94a3b8 !important;
}

.mermaid-fallback {
  font-size: 11px;
  white-space: pre-wrap;
  word-break: break-all;
  color: #64748b;
  padding: 12px;
}
</style>
