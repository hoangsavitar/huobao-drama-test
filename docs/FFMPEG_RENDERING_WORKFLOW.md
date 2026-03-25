# FFmpeg Rendering & Video Processing Workflow

This document details the video merging, clipping, transitions, and duration mathematics handled by the Go backend via FFmpeg (`infrastructure/external/ffmpeg/ffmpeg.go` and `application/services/video_merge_service.go`).

## 1. Core Architecture
The system does not rely on third-party video generation APIs for its final composition. Instead, it aggregates generated video clips locally using FFmpeg. 
- **Wrapper Logic**: High-level Go struct `ffmpeg` executes OS-level `exec.Command("ffmpeg", ...)`.
- **Primary Video Codec**: `libx264` (H.264), optimized broadly for web and mobile.
- **Primary Audio Codec**: `aac`.
- **Quality Preset**: Variable bitrate controlled by `-crf 23` alongside `-preset fast` for an optimal balance of rendering speed and visual fidelity.

## 2. The Merging Process (`MergeVideos`)
When a user requests a video compilation (e.g., matching storyboard clips to form an episode), `application/services/video_merge_service.go` processes sequentially:

### Scene Iteration & Validation
1. Loops through `req.Scenes` to confirm `scene.VideoURL` exists.
2. Formats all inputs into `ffmpeg.VideoClip` structs mapped with:
   - `URL` (Local storage or HTTP URL)
   - `Duration` (Float64 seconds)
   - `StartTime` / `EndTime` (Clipping endpoints)
   - `Transition` (Map of properties for cross-fades)

### Mathematical Accumulation
Prior to FFmpeg invocation, the backend strictly accumulates the expected `totalDuration`:
```go
var totalDuration float64
for _, scene := range scenes {
    totalDuration += scene.Duration
}
```
This forces duration predictability. If a 3rd party AI hallucinates a video that is 7 seconds long instead of the instructed 5 seconds, the system trims it based on the defined Start and End bounds.

## 3. Advanced FFmpeg Filtering
Depending on the instructions embedded within `ffmpeg.VideoClip`, the backend applies specific complex filters:

### Trimming (`-ss` and `-to`)
If a generated clip contains excess padding or needs cutting, the parameters `-ss` (start time) and `-to` (end time) are injected dynamically to extract the precise sub-clip before it hits the merge queue.

### Transitions (`xfade`)
If a `Transition` map is provided, the backend calculates overlapping frames.
- It triggers the `mergeWithXfade(...)` mechanic.
- Uses FFmpeg's `xfade` complex filter to overlay the visual tails and heads of intersecting clips.
- Time mathematically shifts based on the overlap duration, which implicitly adjusts the `totalDuration` output.

## 4. Final Finalization & Database Synchronization
Once `ffmpeg.MergeVideos()` securely writes out the `libx264` `.mp4` file to `data/storage/videos/merged/`:
1. The relative URL route is passed back to `result`.
2. The Database `models.VideoMerge` row updates with the exact final `Duration`.
3. The parent `models.Episode` changes status to `completed` and maps `video_url` to the newly minted compilation.

## 5. Summary Checklists
- **NEVER** trust AI for exact duration metrics post-generation; always use `ffmpeg.GetVideoDuration(absPath)` when ingesting new singles.
- **ALWAYS** retain `-c:v libx264 -preset fast -crf 23` in adjustments to maintain server stability and prevent storage bloat.
- **TRANSITIONS** eat into total duration (e.g., a 1s crossfade between two 5s clips yields a 9s total, not 10s).

## 6. Code Locations
- **FFmpeg Core Wrapper**: `infrastructure/external/ffmpeg/ffmpeg.go`
- **Video Merge Service**: `application/services/video_merge_service.go`
- **Documentation File**: `docs/FFMPEG_RENDERING_WORKFLOW.md`
