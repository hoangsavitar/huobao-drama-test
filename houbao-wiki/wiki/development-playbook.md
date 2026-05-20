# Development Playbook

**Summary**: A guide for developers on extending the system, rendering video, and troubleshooting.

**Sources**: [[raw/DEVELOPMENT_GUIDE.md]]

**Last updated**: 2026-04-23

---

## Production Flow
The end-to-end flow moves from Drama creation to final FFMPEG export.

## Extension Guide
To add new features:
1. Update Models.
2. Update Prompt logic in `prompt_i18n.go`.
3. Create Service logic.
4. Create Handlers and Routes.
5. Update Frontend API and UI.

## Video Composition
The system uses a specific [[ffmpeg-rendering]] workflow to merge individual shot clips into a final episode video.

## Common Issues
Refer to the troubleshooting section for fixes on common bugs like stale prompts or aspect ratio mismatches.

## Related pages
- [[ffmpeg-rendering]]
- [[architecture-overview]]
- [[ai-integration-details]]
