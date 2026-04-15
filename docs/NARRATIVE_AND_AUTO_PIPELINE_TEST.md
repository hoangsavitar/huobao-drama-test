# Narrative + auto pipeline — QA checklist

Manual regression. Graph is derived from `episodes` + `choices` JSON (no extra graph table).

## Narrative generate (Go)

| # | Case | Expected |
|---|------|----------|
| N1 | No text AI, `fallback_stub: false`, no NarrativePackageService path | 503 `narrative: AI service not available...` |
| N2 | Valid text config, `fallback_stub: false` | LLM JSON → normalized graph saved; errors surface on failure |
| N3 | `fallback_stub: true` + LLM failure | 7-node template DAG saved |
| N4 | Empty `user_idea` | OK (uses drama title); graph still requires successful LLM or fallback |
| N5 | Missing drama | 404 |

## Story graph (Episode Management)

| # | Case | Expected |
|---|------|----------|
| G1 | No `narrative_node_id`, no choices | Mermaid shows placeholder / minimal |
| G2 | Branching with `next_episode_id` | Edges between episode nodes |
| G3 | Only `next_narrative_node_id` | Edges if node ids resolve to episodes |

## Full auto production (browser)

| # | Case | Expected |
|---|------|----------|
| P1 | Episode without script | Fails at `validate_script` |
| P2 | Cancel during run | `AbortError` → queue `cancelled`, no further submits |
| P3 | Delete episode before its turn | Log "Skip … removed" |
| P4 | No video model in localStorage | Completes with message to set video model in Episode workflow |
| P5 | One shot missing first frame | Video loop skips that shot; others continue |

Smoke: after generate narrative, open Interactive Play — choices resolve to next episode.
