import type { Episode, EpisodeChoice } from "@/types/drama";

function sanitizeMermaidId(raw: string | number): string {
  const s = String(raw).replace(/[^a-zA-Z0-9_]/g, "_");
  return s.length ? s : "node";
}

function normalizeChoices(ep: Episode): EpisodeChoice[] {
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
}

/** Build node→episode map by narrative_node_id for resolving narrative edges. */
function narrativeNodeMap(episodes: Episode[]): Map<string, Episode> {
  const m = new Map<string, Episode>();
  for (const ep of episodes) {
    const nid = ep.narrative_node_id?.trim();
    if (nid) m.set(nid, ep);
  }
  return m;
}

/**
 * Mermaid flowchart TD source: one node per episode, edges from choices.
 * Labels: Ep{n} · title · narrative id (truncated).
 */
export function buildNarrativeMermaidSource(episodes: Episode[]): string {
  if (!episodes?.length) {
    return "flowchart TD\n  empty[\"No episodes\"]\n";
  }
  const sorted = [...episodes].sort(
    (a, b) => (a.episode_number || 0) - (b.episode_number || 0),
  );
  const byNid = narrativeNodeMap(sorted);
  const lines: string[] = ["flowchart TD"];

  for (const ep of sorted) {
    const id = sanitizeMermaidId(`E${ep.id}`);
    const nn = ep.narrative_node_id?.trim() || "—";
    const title = (ep.title || "Untitled").replace(/"/g, "'").slice(0, 42);
    const entry = ep.is_entry ? " ⭐" : "";
    lines.push(`  ${id}["Ep${ep.episode_number}${entry}: ${title}<br/>${nn}"]`);
  }

  const seen = new Set<string>();
  for (const ep of sorted) {
    const from = sanitizeMermaidId(`E${ep.id}`);
    const choices = normalizeChoices(ep);
    for (const ch of choices) {
      let targetEp: Episode | undefined;
      if (ch.next_episode_id != null) {
        targetEp = sorted.find((e) => Number(e.id) === Number(ch.next_episode_id));
      } else if (ch.next_narrative_node_id?.trim()) {
        targetEp = byNid.get(ch.next_narrative_node_id.trim());
      }
      if (!targetEp) continue;
      const to = sanitizeMermaidId(`E${targetEp.id}`);
      const key = `${from}->${to}`;
      if (seen.has(key)) continue;
      seen.add(key);
      const lbl = (ch.label || "→").replace(/"/g, "'").slice(0, 28);
      lines.push(`  ${from} -->|"${lbl}"| ${to}`);
    }
  }

  if (lines.length === 1) {
    lines.push('  linear["No branching choices — linear or empty choices JSON"]');
  }

  return lines.join("\n");
}
