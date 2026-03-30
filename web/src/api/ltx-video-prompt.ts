import request from '../utils/request'

export interface BatchGenerateLtxVideoPromptsRequest {
  storyboard_ids: Array<number | string>
  model?: string
}

export interface BatchGenerateLtxVideoPromptsResponse {
  task_id: string
  status: string
  message: string
}

export const ltxVideoPromptAPI = {
  batchGenerateLtxVideoPrompts(
    episodeId: string | number,
    storyboardIds: Array<number | string>,
    model?: string,
  ) {
    const payload: BatchGenerateLtxVideoPromptsRequest = {
      storyboard_ids: storyboardIds,
      ...(model ? { model } : {}),
    }

    return request.post<BatchGenerateLtxVideoPromptsResponse>(
      `/episodes/${episodeId}/ltx-video-prompts/batch`,
      payload,
    )
  },
}

