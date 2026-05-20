<template>
  <div class="page-container">
    <AppHeader :fixed="false" :show-logo="false">
      <template #left>
        <el-button text class="back-btn" @click="$router.back()">
          <span>Back</span>
        </el-button>
        <h1 class="page-title-h1">Interactive play</h1>
      </template>
    </AppHeader>

    <div v-if="currentEpisode" class="play-wrap">
      <p class="ep-label">
        Ep {{ currentEpisode.episode_number }} · {{ currentEpisode.title }}
        <el-tag v-if="currentEpisode.narrative_node_id" size="small" style="margin-left: 8px">
          {{ currentEpisode.narrative_node_id }}
        </el-tag>
      </p>
      <video
        v-if="videoSrc"
        ref="videoRef"
        :key="String(currentEpisode.id)"
        :src="videoSrc"
        controls
        playsinline
        class="play-video"
        @ended="onVideoEnded"
      />
      <el-empty
        v-else
        description="No merged video yet — finish production (merge) for this episode."
      />
      <div v-if="showChoices && branchChoices.length" class="choices">
        <el-button
          v-for="(ch, idx) in branchChoices"
          :key="idx"
          type="primary"
          class="choice-btn"
          @click="pickChoice(ch)"
        >
          {{ ch.label }}
        </el-button>
      </div>
    </div>
    <el-empty v-else description="Loading…" />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { useRoute } from "vue-router";
import { dramaAPI } from "@/api/drama";
import type { Drama, Episode, EpisodeChoice } from "@/types/drama";
import { AppHeader } from "@/components/common";
import { ElMessage } from "element-plus";

const route = useRoute();
const drama = ref<Drama | null>(null);
const currentEpisode = ref<Episode | null>(null);
const showChoices = ref(false);
const videoRef = ref<HTMLVideoElement | null>(null);

const normalizeChoices = (ep: Episode): EpisodeChoice[] => {
  const c = ep.choices as unknown;
  if (!c) return [];
  if (Array.isArray(c)) return c as EpisodeChoice[];
  if (typeof c === "string") {
    try {
      const p = JSON.parse(c);
      return Array.isArray(p) ? p : [];
    } catch {
      return [];
    }
  }
  return [];
};

const branchChoices = computed(() =>
  currentEpisode.value ? normalizeChoices(currentEpisode.value) : [],
);

const videoSrc = computed(() => {
  const u = currentEpisode.value?.video_url?.trim();
  if (!u) return "";
  return u;
});

const pickStartEpisode = (d: Drama): Episode | null => {
  const eps = d.episodes || [];
  if (!eps.length) return null;
  const entry = eps.find((e) => e.is_entry);
  if (entry) return entry;
  return [...eps].sort((a, b) => a.episode_number - b.episode_number)[0];
};

const load = async () => {
  const id = route.params.id as string;
  const data = await dramaAPI.get(id);
  drama.value = data;
  currentEpisode.value = pickStartEpisode(data);
  showChoices.value = false;
};

const onVideoEnded = () => {
  if (branchChoices.value.length) showChoices.value = true;
};

const pickChoice = (ch: EpisodeChoice) => {
  if (!drama.value?.episodes || !currentEpisode.value) return;
  const nextId = ch.next_episode_id;
  if (nextId == null) {
    ElMessage.warning("Missing next_episode_id for this choice.");
    return;
  }
  const next = drama.value.episodes.find(
    (e) => Number(e.id) === Number(nextId) || String(e.id) === String(nextId),
  );
  if (!next) {
    ElMessage.error("Next episode not found");
    return;
  }
  showChoices.value = false;
  currentEpisode.value = next;
  requestIframe(() => {
    const v = videoRef.value;
    if (v) {
      v.load();
      v.play().catch(() => {});
    }
  });
};

function requestIframe(fn: () => void) {
  requestAnimationFrame(() => requestAnimationFrame(fn));
}

onMounted(() => {
  load().catch((e: unknown) => {
    const msg = e instanceof Error ? e.message : "Load failed";
    ElMessage.error(msg);
  });
});
</script>

<style scoped>
.play-wrap {
  max-width: 960px;
  margin: 24px auto;
  padding: 0 16px 32px;
}
.play-video {
  width: 100%;
  border-radius: 8px;
  background: #000;
}
.choices {
  margin-top: 16px;
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
}
.choice-btn {
  min-width: 160px;
}
.ep-label {
  margin: 0 0 12px;
  font-size: 15px;
}
.page-title-h1 {
  display: inline;
  font-size: 1.25rem;
  font-weight: 600;
  margin-left: 8px;
}
</style>
