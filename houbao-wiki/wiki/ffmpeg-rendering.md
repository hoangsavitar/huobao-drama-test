# FFMPEG Rendering

**Summary**: The technical workflow for merging individual shot videos into a complete episode.

**Sources**: [[raw/DEVELOPMENT_GUIDE.md]]

**Last updated**: 2026-04-23

---

The system calls the FFMPEG CLI directly from the backend (`infrastructure/external/ffmpeg/`).

## Merge Logic
- **Duration Enforcement**: FFMPEG uses `-ss` and `-to` to match the configured duration for each shot.
- **Transitions**: Uses the `xfade` filter. Note that transitions reduce the total video duration.
- **Codecs**: Standardized to `libx264` and `aac`.

## Workflow
1. Collect all `video_url` links for an Episode.
2. Apply duration math.
3. Apply transitions if specified.
4. Execute merge command and return the final video file.

## Related pages
- [[development-playbook]]
- [[architecture-overview]]
