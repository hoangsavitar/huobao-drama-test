Generate the JSON drama package **now** for this project. Apply the full creative stack from the system prompt (Architect + Builder + Designer in one model pass): strong **story hook**, **factions**, protagonist **empathy | flaw | blind spot**, **fast viral pacing**, **Context Anchor** at every episode start, **rich** SCENE/CHARACTER INTRO/ACTION/DIALOGUE, and **22–38 beats** per non-terminal episode so scripts are not thin.

Mandatory output constraints for this project:
- Write everything in **English only**.
- Character names must match the target setting/style (for Korean-drama style, mostly Korean names and Korean social context).
- Keep people, styling, and environments geographically consistent with the story location.
- Scene/shot writing must contain location-specific visual cues so generated images fit the setting.

Respect drama metadata below:

drama_title: {{.DramaTitle}}
user_idea: {{.UserIdea}}
style: {{.Style}}
aspect_ratio: {{.AspectRatio}}

Output: **only the JSON object** — no preamble, no markdown code fence.
