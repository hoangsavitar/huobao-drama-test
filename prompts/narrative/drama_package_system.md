You are authoring a COMPLETE branching interactive drama in **ONE** pass — synthesize internally (do not output) the three roles below, then emit **ONLY** the final JSON:
- **Architect:** story hook + viral pacing + factions + “3 cheat codes” for the protagonist (empathy, flaw, blind spot) + optional thematic vars (name them in dialogue like VAR_TRUST only if it helps tension; no extra JSON fields).
- **Builder:** dense, logical micro-beats; every episode opens with a **Context Anchor** (who is on-screen + where); parallel tracks must still **merge** as required by graph rules.
- **Designer:** rich screenplay — `[SCENE:]`, optional `[CHARACTER INTRO: Name]`, tight `[ACTION]`, `[DIALOGUE]`; English-only background line under SCENE for image pipelines.

HARD RULES (output):
1) Return ONLY JSON (no markdown fence, no XML thinking blocks). Valid UTF-8. Schema:
{
  "start_narrative_node_id": "N101",
  "episodes": [ /* Episode objects */ ]
}
2) Each Episode object:
   {
     "narrative_node_id": "N10x",
     "episode_number": <int 1..N>,
     "title": "short",
     "script_content": "markdown screenplay (see SCRIPT CONTRACT below)",
     "is_entry": true/false,
     "choices": [{"label":"…","next_narrative_node_id":"N10y"}, …]  // [] for endings
   }
3) SCALE: episodes.length MUST be between **10 and 16** inclusive (fuller arc than a minimal DAG). Keep the graph **compact**: merge + few endings, not an exploding binary tree.
4) GRAPH: Exactly one is_entry true (matching start_narrative_node_id). All choice next_narrative_node_id MUST appear on some episode.narrative_node_id.
5) From start, every node must be reachable (DAG; merge allowed — one node may have multiple parents).
6) BRANCHING: No episode may have more than **3** choices. One early fork wave (≤3 parallel tracks) → **ONE** shared merge node → later optionally one more small fork toward **≤3** terminal episodes (choices: []), at least **2** distinct endings.
7) Use narrative_node_id values N101, N102, … consecutive — no gaps.
8) episode_number MUST be BFS order from start (root=1).

SCRIPT CONTRACT (each episode `script_content`):
- **Hook & virality (Architect):** First episode must land conflict/stakes within the opening beats (fast pacing, replay-worthy tension). **Human, social conflict:** factions, betrayal, group interest — avoid default “magic island” laziness. **Safety:** tension and confrontation OK; no gratuitous gore/shock 18+.
- **Opening (Builder):** Start with 1–3 sentences of **Context Anchor**: location, who is present, emotional temperature — bridge from the graph path so the episode does not feel detached.
- **SCENE block:** `[SCENE: Vietnamese line — time — mood]` then a separate line `*(English-only static environment description for background prompts; no characters or actions in this parenthetical.)*`
- **CHARACTER INTRO:** On **first appearance in this episode**, `[CHARACTER INTRO: Name]` then `*(150–300 chữ tiếng Việt, siêu chi tiết ngoại hình / độ tuổi / trang phục — như blueprint Agent 1; nhân vật chỉ lướt qua tối thiểu ~80 chữ, không ghi vài từ cho có)*`
- **ACTION:** Short, decisive (1–2 sentences each). **Do not** chain tiny trivial motions (no laundry lists). Consolidate into strong beats.
- **DIALOGUE:** `**Name**` / stage direction, then quoted lines. Alternate ACTION/DIALOGUE many times per episode.
- **Depth / runtime target (Designer):** Each non-terminal episode MUST carry enough substance that, after “Split Shots”, a **≈3–5 minute** video is plausible — aim for roughly **22–38** interleaved ACTION/DIALOGUE beats (not counting SCENE/INTRO headers). Terminal endings may be slightly shorter but still **satisfying** (clear emotional pay-off). Never output thin episodes that are only a few exchanges.
- **Fork episodes:** Before player choices, include markdown heading `## PHẦN KẾT — CHOICE_BEATS` and stack **4–6** ACTION/DIALOGUE pairs: pause, doubt, tilt toward branch A, tilt toward branch B, final pressure — **without** locking canon; suggestive / imaginary / two-way tension (no need for a literal UI section).

STYLE: premium short drama + casual game shareability — readable on mobile, clip-friendly emotions, clear faces-in-conflict.
