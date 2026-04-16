import { dramaAPI } from "@/api/drama";
import { generationAPI } from "@/api/generation";
import { characterLibraryAPI } from "@/api/character-library";
import { imageAPI } from "@/api/image";
import { ltxVideoPromptAPI } from "@/api/ltx-video-prompt";
import { videoAPI } from "@/api/video";
import { aiAPI } from "@/api/ai";
import {
  generateFirstFrame,
  getStoryboardFramePrompts,
} from "@/api/frame";
import { taskAPI } from "@/api/task";
import type { Drama, Episode } from "@/types/drama";
import type { ImageGeneration } from "@/types/image";
import type { GenerateVideoRequest } from "@/types/video";

export type PipelineStep =
  | "validate_script"
  | "extract"
  | "batch_char_images"
  | "batch_scene_images"
  | "split_storyboard"
  | "batch_first_frame_prompts"
  | "batch_shot_images"
  | "batch_ltx_prompts"
  | "batch_videos";

export type PipelineQueueStatus =
  | "idle"
  | "running"
  | "cancelled"
  | "failed"
  | "completed";

export interface PipelineResult {
  ok: boolean;
  failedStep?: PipelineStep;
  episodeId?: string;
  episodeNumber?: number;
  message?: string;
}

export interface PipelineModels {
  textModel: string;
  imageModel: string;
  videoModel: string;
}

export interface StepLogEntry {
  step: PipelineStep;
  startedAt: string;
  endedAt?: string;
  status: "ok" | "fail" | "skipped";
  message?: string;
}

export function getPipelineModelsFromStorage(dramaId: string): PipelineModels {
  return {
    textModel: localStorage.getItem(`ai_text_model_${dramaId}`) || "",
    imageModel: localStorage.getItem(`ai_image_model_${dramaId}`) || "",
    videoModel: localStorage.getItem(`ai_video_model_${dramaId}`) || "",
  };
}

export async function getPipelineModelsWithFallback(
  dramaId: string,
): Promise<PipelineModels> {
  const models = getPipelineModelsFromStorage(dramaId);
  if (models.videoModel?.trim()) return models;

  try {
    const configs = await aiAPI.list("video");
    const activeConfigs = configs.filter((c) => c.is_active);
    const allModels = activeConfigs
      .flatMap((config) => {
        const modelList = Array.isArray(config.model)
          ? config.model
          : [config.model];
        return modelList
          .filter(Boolean)
          .map((modelName) => ({
            modelName,
            priority: config.priority || 0,
          }));
      })
      .sort((a, b) => b.priority - a.priority);

    const fallbackVideoModel = allModels[0]?.modelName || "";
    if (fallbackVideoModel) {
      models.videoModel = fallbackVideoModel;
      localStorage.setItem(`ai_video_model_${dramaId}`, fallbackVideoModel);
    }
  } catch {
    // Keep storage values if AI config fetch fails.
  }

  return models;
}

function sleep(ms: number) {
  return new Promise((r) => setTimeout(r, ms));
}

function throwIfAborted(signal?: AbortSignal) {
  if (signal?.aborted) {
    throw new DOMException("Aborted", "AbortError");
  }
}

function extractProviderFromModel(modelName: string): string {
  if (modelName.startsWith("doubao-") || modelName.startsWith("seedance")) {
    return "doubao";
  }
  if (modelName.startsWith("runway")) return "runway";
  if (modelName.startsWith("pika")) return "pika";
  if (
    modelName.startsWith("MiniMax-") ||
    modelName.toLowerCase().startsWith("minimax") ||
    modelName.startsWith("hailuo")
  ) {
    return "minimax";
  }
  if (modelName.startsWith("sora")) return "openai";
  if (modelName.startsWith("kling")) return "kling";
  return "doubao";
}

function storyboardVideoPrompt(sb: Record<string, unknown>): string {
  const ltx = String(sb?.ltx_video_prompt ?? "").trim();
  const vp = String(sb?.video_prompt ?? "").trim();
  const act = String(sb?.action ?? "").trim();
  return (ltx || vp || act || "").trim();
}

async function pollExtractTask(
  taskId: string,
  type: "character" | "background",
  dramaId: string,
  episodeId: number | undefined,
  signal?: AbortSignal,
): Promise<void> {
  const maxAttempts = 60;
  const interval = 2000;
  for (let i = 0; i < maxAttempts; i++) {
    throwIfAborted(signal);
    await sleep(interval);
    const task = await generationAPI.getTaskStatus(taskId);
    if (task.status === "completed") {
      if (type === "character" && task.result) {
        const result =
          typeof task.result === "string"
            ? JSON.parse(task.result)
            : task.result;
        if (result?.characters?.length) {
          await dramaAPI.saveCharacters(
            dramaId,
            result.characters,
            episodeId != null ? String(episodeId) : undefined,
          );
        }
      }
      return;
    }
    if (task.status === "failed") {
      throw new Error(
        task.error ||
          (type === "character"
            ? "Character generation failed"
            : "Background extraction failed"),
      );
    }
  }
  throw new Error(
    type === "character"
      ? "Character generation timeout"
      : "Background extraction timeout",
  );
}

async function waitForStoryboardTask(
  taskId: string,
  signal?: AbortSignal,
): Promise<void> {
  for (let i = 0; i < 120; i++) {
    throwIfAborted(signal);
    await sleep(2000);
    const task = await generationAPI.getTaskStatus(taskId);
    if (task.status === "completed") return;
    if (task.status === "failed") {
      throw new Error(task.error || "Storyboard split failed");
    }
  }
  throw new Error("Storyboard split timeout");
}

async function pollTaskUntilDoneStrict(
  taskId: string,
  signal?: AbortSignal,
): Promise<void> {
  for (let i = 0; i < 60; i++) {
    throwIfAborted(signal);
    await sleep(3000);
    const task = await taskAPI.getStatus(taskId);
    if (task.status === "completed") return;
    if (task.status === "failed") {
      throw new Error(task.error || "Async task failed");
    }
  }
  throw new Error("Async task timeout");
}

export interface RunEpisodePipelineOptions {
  dramaId: string;
  episodeId: string;
  episodeNumber: number;
  models: PipelineModels;
  signal?: AbortSignal;
  onStep?: (entry: StepLogEntry) => void;
}

function logStep(
  onStep: ((e: StepLogEntry) => void) | undefined,
  partial: Omit<StepLogEntry, "startedAt" | "endedAt" | "status"> & {
    status?: StepLogEntry["status"];
    message?: string;
  },
  start: string,
  end?: string,
  status: StepLogEntry["status"] = "ok",
) {
  onStep?.({
    step: partial.step,
    startedAt: start,
    endedAt: end,
    status,
    message: partial.message,
  });
}

/**
 * Runs the same sequence as EpisodeWorkflow: extract → char/scene images → split →
 * first-frame prompts → shot images → LTX prompts → per-shot video.
 */
export async function runFullEpisodePipeline(
  opts: RunEpisodePipelineOptions,
): Promise<PipelineResult> {
  const { dramaId, episodeId, episodeNumber, models, signal, onStep } = opts;
  const now = () => new Date().toISOString();

  const fail = (
    step: PipelineStep,
    message: string,
  ): PipelineResult => ({
    ok: false,
    failedStep: step,
    episodeId,
    episodeNumber,
    message,
  });

  try {
    throwIfAborted(signal);

    let drama: Drama = await dramaAPI.get(dramaId);
    let ep = drama.episodes?.find(
      (e) => String(e.id) === String(episodeId),
    ) as Episode | undefined;
    if (!ep) {
      return fail("validate_script", "Episode not found (may have been deleted).");
    }

    const script = (ep.script_content || "").trim();
    const t0 = now();
    if (!script) {
      logStep(
        onStep,
        {
          step: "validate_script",
          message: "Missing script_content",
        },
        t0,
        now(),
        "fail",
      );
      return fail("validate_script", "Episode has no script — add content first.");
    }
    logStep(onStep, { step: "validate_script", message: "OK" }, t0, now());

    /* extract */
    const t1 = now();
    try {
      const epNumId = Number(episodeId);
      const [charTask, bgTask] = await Promise.all([
        generationAPI.generateCharacters({
          drama_id: dramaId,
          episode_id: epNumId,
          outline: script,
          count: 0,
          model: models.textModel || undefined,
        }),
        dramaAPI.extractBackgrounds(
          String(episodeId),
          models.textModel || undefined,
        ),
      ]);
      await Promise.all([
        pollExtractTask(charTask.task_id, "character", dramaId, epNumId, signal),
        pollExtractTask(bgTask.task_id, "background", dramaId, epNumId, signal),
      ]);
      drama = await dramaAPI.get(dramaId);
      ep = drama.episodes?.find((e) => String(e.id) === String(episodeId));
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : String(e);
      logStep(
        onStep,
        { step: "extract", message: msg },
        t1,
        now(),
        "fail",
      );
      return fail("extract", msg);
    }
    logStep(onStep, { step: "extract", message: "OK" }, t1, now());

    throwIfAborted(signal);

    /* batch character images */
    const t2 = now();
    const charIds =
      ep?.characters?.map((c) => c.id).filter(Boolean) || [];
    if (charIds.length) {
      try {
        await characterLibraryAPI.batchGenerateCharacterImages(
          charIds.map(String),
          models.imageModel || undefined,
        );
      } catch (e: unknown) {
        const msg = e instanceof Error ? e.message : String(e);
        logStep(
          onStep,
          { step: "batch_char_images", message: msg },
          t2,
          now(),
          "fail",
        );
        return fail("batch_char_images", msg);
      }
    } else {
      logStep(
        onStep,
        { step: "batch_char_images", message: "No characters — skipped" },
        t2,
        now(),
        "skipped",
      );
    }
    logStep(onStep, { step: "batch_char_images", message: "OK" }, t2, now());

    drama = await dramaAPI.get(dramaId);
    ep = drama.episodes?.find((e) => String(e.id) === String(episodeId));

    /* batch scene images */
    const t3 = now();
    const sceneIds = ep?.scenes?.map((s) => s.id).filter(Boolean) || [];
    if (sceneIds.length) {
      try {
        await Promise.allSettled(
          sceneIds.map((sid) =>
            dramaAPI.generateSceneImage({
              scene_id: Number(sid),
              model: models.imageModel || undefined,
            }),
          ),
        );
      } catch (e: unknown) {
        const msg = e instanceof Error ? e.message : String(e);
        logStep(
          onStep,
          { step: "batch_scene_images", message: msg },
          t3,
          now(),
          "fail",
        );
        return fail("batch_scene_images", msg);
      }
    } else {
      logStep(
        onStep,
        { step: "batch_scene_images", message: "No scenes — skipped" },
        t3,
        now(),
        "skipped",
      );
    }
    logStep(onStep, { step: "batch_scene_images", message: "OK" }, t3, now());

    /* split storyboard */
    const t4 = now();
    try {
      const res = await generationAPI.generateStoryboard(
        String(episodeId),
        models.textModel || undefined,
      );
      await waitForStoryboardTask(res.task_id, signal);
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : String(e);
      logStep(
        onStep,
        { step: "split_storyboard", message: msg },
        t4,
        now(),
        "fail",
      );
      return fail("split_storyboard", msg);
    }
    logStep(onStep, { step: "split_storyboard", message: "OK" }, t4, now());

    drama = await dramaAPI.get(dramaId);
    ep = drama.episodes?.find((e) => String(e.id) === String(episodeId));
    const storyboards = ep?.storyboards || [];
    const sbIds = storyboards.map((s) => Number(s.id)).filter((n) => !Number.isNaN(n));
    if (!sbIds.length) {
      return fail("split_storyboard", "No storyboards after split.");
    }

    throwIfAborted(signal);

    /* first frame prompts */
    const t5 = now();
    try {
      const results = await Promise.allSettled(
        sbIds.map((id) => generateFirstFrame(id)),
      );
      const taskIds: string[] = [];
      for (const r of results) {
        if (r.status === "fulfilled" && r.value?.task_id) {
          taskIds.push(r.value.task_id);
        }
      }
      await Promise.all(
        taskIds.map((tid) => pollTaskUntilDoneStrict(tid, signal)),
      );
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : String(e);
      logStep(
        onStep,
        { step: "batch_first_frame_prompts", message: msg },
        t5,
        now(),
        "fail",
      );
      return fail("batch_first_frame_prompts", msg);
    }
    logStep(
      onStep,
      { step: "batch_first_frame_prompts", message: "OK" },
      t5,
      now(),
    );

    throwIfAborted(signal);

    /* shot images */
    const t6 = now();
    try {
      for (const storyboardId of sbIds) {
        throwIfAborted(signal);
        const sb = storyboards.find((s) => Number(s.id) === storyboardId) as
          | Record<string, unknown>
          | undefined;
        if (!sb) continue;
        const fpData = await getStoryboardFramePrompts(storyboardId);
        const fp = fpData.frame_prompts?.find((p) => p.frame_type === "first");
        if (!fp?.prompt) continue;
        const referenceImages: string[] = [];
        const bg = sb.background as { local_path?: string; image_url?: string } | undefined;
        if (bg?.local_path) referenceImages.push(bg.local_path);
        else if (bg?.image_url) referenceImages.push(bg.image_url);
        const chars = sb.characters as Array<{ local_path?: string; image_url?: string }> | undefined;
        if (Array.isArray(chars)) {
          chars.forEach((char) => {
            if (char.local_path) referenceImages.push(char.local_path);
            else if (char.image_url) referenceImages.push(char.image_url);
          });
        }
        await imageAPI.generateImage({
          drama_id: dramaId,
          prompt: fp.prompt,
          storyboard_id: storyboardId,
          image_type: "storyboard",
          frame_type: "first",
          reference_images:
            referenceImages.length > 0 ? referenceImages : undefined,
          model: models.imageModel || undefined,
        });
      }
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : String(e);
      logStep(
        onStep,
        { step: "batch_shot_images", message: msg },
        t6,
        now(),
        "fail",
      );
      return fail("batch_shot_images", msg);
    }
    logStep(onStep, { step: "batch_shot_images", message: "OK" }, t6, now());

    throwIfAborted(signal);

    /* LTX batch */
    const t7 = now();
    try {
      const ltxRes = await ltxVideoPromptAPI.batchGenerateLtxVideoPrompts(
        String(episodeId),
        sbIds,
        models.textModel || undefined,
      );
      if (ltxRes?.task_id) {
        await pollTaskUntilDoneStrict(ltxRes.task_id, signal);
      }
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : String(e);
      logStep(
        onStep,
        { step: "batch_ltx_prompts", message: msg },
        t7,
        now(),
        "fail",
      );
      return fail("batch_ltx_prompts", msg);
    }
    logStep(onStep, { step: "batch_ltx_prompts", message: "OK" }, t7, now());

    drama = await dramaAPI.get(dramaId);
    ep = drama.episodes?.find((e) => String(e.id) === String(episodeId));
    const sbAfter = ep?.storyboards || [];

    /* videos */
    const t8 = now();
    if (!models.videoModel?.trim()) {
      logStep(
        onStep,
        {
          step: "batch_videos",
          message: "No video model in storage — skipped",
        },
        t8,
        now(),
        "skipped",
      );
      return {
        ok: true,
        episodeId,
        episodeNumber,
        message:
          "Pipeline finished except video: select a video model in Episode workflow Text/Image Config once, then re-run or batch video there.",
      };
    }

    try {
      const byId = new Map<number, Record<string, unknown>>(
        sbAfter.map((s) => [Number(s.id), s as Record<string, unknown>]),
      );
      let submitted = 0;
      let skippedNoPrompt = 0;
      let skippedNoFirstFrame = 0;
      for (const storyboardId of sbIds) {
        throwIfAborted(signal);
        const sb = byId.get(storyboardId);
        const prompt = sb ? storyboardVideoPrompt(sb) : "";
        if (!prompt || prompt.length < 5) {
          skippedNoPrompt++;
          continue;
        }

        let first: ImageGeneration | undefined;
        try {
          // Batch shot images may still be processing. Wait a bit for first-frame completion.
          const maxChecks = 20; // ~60s
          for (let check = 0; check < maxChecks; check++) {
            const imgRes = await imageAPI.listImages({
              storyboard_id: storyboardId,
              frame_type: "first",
              page: 1,
              page_size: 30,
            });
            first = imgRes.items?.find(
              (i) =>
                i.status === "completed" && (i.image_url || i.local_path),
            );
            if (first) break;
            await sleep(3000);
          }
        } catch {
          first = undefined;
        }
        if (!first) {
          skippedNoFirstFrame++;
          continue;
        }

        const duration = Math.min(
          10,
          Math.max(4, Math.round(Number(sb?.duration) || 5)),
        );

        const req: GenerateVideoRequest & Record<string, unknown> = {
          drama_id: dramaId,
          storyboard_id: storyboardId,
          prompt,
          duration,
          provider: extractProviderFromModel(models.videoModel),
          model: models.videoModel,
          reference_mode: "single",
          aspect_ratio: drama.aspect_ratio || "16:9",
          image_gen_id: first.id,
        };
        if (first.local_path) {
          req.image_local_path = first.local_path;
        } else if (first.image_url) {
          req.image_url = first.image_url;
        }
        await videoAPI.generateVideo(req);
        submitted++;
      }

      const summaryMsg = `submitted=${submitted}, skipped_no_prompt=${skippedNoPrompt}, skipped_no_first_frame=${skippedNoFirstFrame}`;
      if (submitted === 0) {
        logStep(
          onStep,
          {
            step: "batch_videos",
            message: `No video API request submitted (${summaryMsg})`,
          },
          t8,
          now(),
          "fail",
        );
        return fail(
          "batch_videos",
          `No video API request submitted (${summaryMsg})`,
        );
      }
      logStep(
        onStep,
        {
          step: "batch_videos",
          message: `OK (${summaryMsg})`,
        },
        t8,
        now(),
      );
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : String(e);
      logStep(
        onStep,
        { step: "batch_videos", message: msg },
        t8,
        now(),
        "fail",
      );
      return fail("batch_videos", msg);
    }

    return { ok: true, episodeId, episodeNumber, message: "Full pipeline submitted." };
  } catch (e: unknown) {
    if (e instanceof DOMException && e.name === "AbortError") {
      return {
        ok: false,
        failedStep: undefined,
        episodeId,
        episodeNumber,
        message: "Cancelled",
      };
    }
    const msg = e instanceof Error ? e.message : String(e);
    return {
      ok: false,
      episodeId,
      episodeNumber,
      message: msg,
    };
  }
}
