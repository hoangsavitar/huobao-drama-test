import type { Asset } from './asset'

export interface Timeline {
  id: number
  drama_id: number
  episode_id?: number
  name: string
  description?: string
  duration: number
  fps: number
  resolution?: string
  status: TimelineStatus
  tracks?: TimelineTrack[]
  created_at: string
  updated_at: string
}

export type TimelineStatus = 'draft' | 'editing' | 'completed' | 'exporting'

export interface TimelineTrack {
  id: number
  timeline_id: number
  name: string
  type: TrackType
  order: number
  is_locked: boolean
  is_muted: boolean
  volume?: number
  clips?: TimelineClip[]
  created_at: string
}

export type TrackType = 'video' | 'audio' | 'text'

export interface TimelineClip {
  id: number
  track_id: number
  asset_id?: number
  asset?: Asset
  scene_id?: number
  name: string
  start_time: number
  end_time: number
  duration: number
  trim_start?: number
  trim_end?: number
  speed?: number
  volume?: number
  is_muted: boolean
  fade_in?: number
  fade_out?: number
  transition_in_id?: number
  transition_out_id?: number
  in_transition?: ClipTransition
  out_transition?: ClipTransition
  effects?: ClipEffect[]
  created_at: string
}

export interface ClipTransition {
  id: number
  type: TransitionType
  duration: number
  easing?: string
  config?: Record<string, any>
}

export type TransitionType = 'fade' | 'crossfade' | 'slide' | 'wipe' | 'zoom' | 'dissolve'

export interface ClipEffect {
  id: number
  clip_id: number
  type: EffectType
  name: string
  is_enabled: boolean
  order: number
  config?: Record<string, any>
}

export type EffectType = 'filter' | 'color' | 'blur' | 'brightness' | 'contrast' | 'saturation'

export interface CreateTimelineRequest {
  drama_id: number
  episode_id?: number
  name: string
  description?: string
  fps?: number
  resolution?: string
}

export interface UpdateTimelineRequest {
  name?: string
  description?: string
  fps?: number
  resolution?: string
  status?: TimelineStatus
}

export interface CreateTrackRequest {
  name: string
  type: TrackType
  order?: number
  volume?: number
}

export interface UpdateTrackRequest {
  name?: string
  order?: number
  is_locked?: boolean
  is_muted?: boolean
  volume?: number
}

export interface CreateClipRequest {
  track_id: number
  asset_id?: number
  scene_id?: number
  name?: string
  start_time: number
  duration: number
  trim_start?: number
  trim_end?: number
  speed?: number
  volume?: number
  fade_in?: number
  fade_out?: number
}

export interface UpdateClipRequest {
  name?: string
  start_time?: number
  duration?: number
  trim_start?: number
  trim_end?: number
  speed?: number
  volume?: number
  is_muted?: boolean
  fade_in?: number
  fade_out?: number
}

export interface CreateTransitionRequest {
  type: TransitionType
  duration: number
  easing?: string
  config?: Record<string, any>
}

export const TRANSITION_TYPES = [
  { label: 'Fade', value: 'fade' },
  { label: 'Crossfade', value: 'crossfade' },
  { label: 'Slide', value: 'slide' },
  { label: 'Wipe', value: 'wipe' },
  { label: 'Zoom', value: 'zoom' },
  { label: 'Dissolve', value: 'dissolve' }
]

export const EFFECT_TYPES = [
  { label: 'Filter', value: 'filter' },
  { label: 'Color', value: 'color' },
  { label: 'Blur', value: 'blur' },
  { label: 'Brightness', value: 'brightness' },
  { label: 'Contrast', value: 'contrast' },
  { label: 'Saturation', value: 'saturation' }
]
