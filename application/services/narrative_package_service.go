package services

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"regexp"
	"strings"
	"time"

	"github.com/drama-generator/backend/domain/models"
	"github.com/drama-generator/backend/pkg/ai"
	"github.com/drama-generator/backend/pkg/logger"
)

//go:embed prompts/narrative/*.md
var narrativePromptFS embed.FS

// NarrativePackageService builds branching episode packages via in-app text AI + embedded prompts; optional stub when FallbackStub is true.
type NarrativePackageService struct {
	ai           *AIService
	log          *logger.Logger
	fallbackStub bool
}

func NewNarrativePackageService(ai *AIService, log *logger.Logger, fallbackStub bool) *NarrativePackageService {
	return &NarrativePackageService{ai: ai, log: log, fallbackStub: fallbackStub}
}

func isTransientModelError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "status 503") ||
		strings.Contains(msg, "status 429") ||
		strings.Contains(msg, "unavailable") ||
		strings.Contains(msg, "high demand") ||
		strings.Contains(msg, "resource_exhausted")
}

func (s *NarrativePackageService) generateNarrativeText(userBlock, systemPrompt string, maxTokens int) (string, error) {
	maxAttempts := 5
	baseDelay := 2 * time.Second

	var lastErr error

	// Thử sử dụng Pro model trước
	if storyClient, getErr := s.ai.GetAIClientForModel("text", "gemini-2.5-pro"); getErr == nil {
		for attempt := 1; attempt <= maxAttempts; attempt++ {
			raw, err := storyClient.GenerateText(userBlock, systemPrompt, ai.WithMaxTokens(maxTokens))
			if err == nil {
				return raw, nil
			}

			lastErr = err
			if !isTransientModelError(err) {
				// Không phải lỗi transient (lỗi logic/prompt), trả về lỗi luôn không retry
				break
			}

			if attempt < maxAttempts {
				delay := baseDelay * time.Duration(1<<(attempt-1)) // 2s, 4s, 8s, 16s...
				if s.log != nil {
					s.log.Warnw(fmt.Sprintf("narrative: pro model transient error (attempt %d/%d), retrying in %v...", attempt, maxAttempts, delay), "error", err)
				}
				time.Sleep(delay)
			}
		}
		
		if s.log != nil && lastErr != nil {
			s.log.Warnw("narrative: preferred pro model failed after retries; falling back to default text client", "error", lastErr)
		}
	} else if s.log != nil {
		s.log.Warnw("narrative: preferred pro model not configured; using default text client", "error", getErr)
	}

	// Thử sử dụng Default model làm fallback
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		raw, err := s.ai.GenerateText(userBlock, systemPrompt, ai.WithMaxTokens(maxTokens))
		if err == nil {
			return raw, nil
		}

		lastErr = err
		if !isTransientModelError(err) {
			break
		}

		if attempt < maxAttempts {
			delay := baseDelay * time.Duration(1<<(attempt-1))
			if s.log != nil {
				s.log.Warnw(fmt.Sprintf("narrative: default model transient error (attempt %d/%d), retrying in %v...", attempt, maxAttempts, delay), "error", err)
			}
			time.Sleep(delay)
		}
	}

	return "", fmt.Errorf("failed to generate text after %d attempts: %w", maxAttempts, lastErr)
}

func (s *NarrativePackageService) failOrStub(idea, dramaTitle, msg string, err error) (*NarrativeDramaPackage, error) {
	if s.fallbackStub {
		if err != nil {
			s.log.Warnw(msg, "error", err)
		} else {
			s.log.Warnw(msg)
		}
		return BuildStubNarrativeDramaPackage(idea, dramaTitle), nil
	}
	if err != nil {
		return nil, fmt.Errorf("%s: %w", msg, err)
	}
	return nil, fmt.Errorf("%s", msg)
}

// BuildPackage calls the configured text model with prompts from markdown assets. If FallbackStub is false, failures return errors (no silent stub).
func (s *NarrativePackageService) BuildPackage(userIdea string, drama models.Drama) (*NarrativeDramaPackage, error) {
	idea := strings.TrimSpace(userIdea)
	if idea == "" {
		idea = strings.TrimSpace(drama.Title)
	}
	if idea == "" {
		return nil, fmt.Errorf("user_idea and drama title are empty")
	}

	system, err := narrativePromptFS.ReadFile("prompts/narrative/drama_package_system.md")
	if err != nil {
		return s.failOrStub(idea, drama.Title, "narrative: read system prompt", err)
	}
	userTplRaw, err := narrativePromptFS.ReadFile("prompts/narrative/drama_package_user.md")
	if err != nil {
		return s.failOrStub(idea, drama.Title, "narrative: read user prompt template", err)
	}

	tpl, err := template.New("user").Parse(string(userTplRaw))
	if err != nil {
		return s.failOrStub(idea, drama.Title, "narrative: parse user template", err)
	}
	var userBuf bytes.Buffer
	_ = tpl.Execute(&userBuf, struct {
		DramaTitle, UserIdea, Style, AspectRatio string
	}{
		DramaTitle:  drama.Title,
		UserIdea:    idea,
		Style:       drama.Style,
		AspectRatio: drama.AspectRatio,
	})
	userBlock := userBuf.String()

	_, err = s.ai.GetAIClient("text")
	if err != nil {
		return s.failOrStub(idea, drama.Title, "narrative: no text AI client configured (Settings > AI Config)", err)
	}

	s.log.Infow("narrative: text model request started (10–16 long episodes can take many minutes; check provider quota if it seems stuck)")
	// Story generator: prefer Pro model; fallback to default text routing (Lite-first).
	if storyClient, getErr := s.ai.GetAIClientForModel("text", "gemini-2.5-pro"); getErr == nil {
		raw, err := storyClient.GenerateText(userBlock, string(system), ai.WithMaxTokens(32768))
		if err != nil {
			return s.failOrStub(idea, drama.Title, "narrative: text generation failed", err)
		}
		raw = stripJSONFence(raw)
		var pkg NarrativeDramaPackage
		if err := json.Unmarshal([]byte(raw), &pkg); err != nil {
			return s.failOrStub(idea, drama.Title, "narrative: model output is not valid JSON", err)
		}

		norm := normalizeNarrativeGraph(&pkg)
		if norm == nil || len(norm.Episodes) < 10 {
			return s.failOrStub(idea, drama.Title, "narrative: graph invalid after normalize or fewer than 10 episodes (prompt expects 10–16)", nil)
		}
		return norm, nil
	}

	raw, err := s.ai.GenerateText(userBlock, string(system), ai.WithMaxTokens(32768))
	if err != nil {
		return s.failOrStub(idea, drama.Title, "narrative: text generation failed", err)
	}

	raw = stripJSONFence(raw)
	var pkg NarrativeDramaPackage
	if err := json.Unmarshal([]byte(raw), &pkg); err != nil {
		return s.failOrStub(idea, drama.Title, "narrative: model output is not valid JSON", err)
	}

	norm := normalizeNarrativeGraph(&pkg)
	if norm == nil || len(norm.Episodes) < 10 {
		return s.failOrStub(idea, drama.Title, "narrative: graph invalid after normalize or fewer than 10 episodes (prompt expects 10–16)", nil)
	}
	return norm, nil
}

type Agent1Character struct {
	Name            string `json:"name"`
	Role            string `json:"role"`
	Description     string `json:"description"`
	Personality     string `json:"personality"`
	Appearance      string `json:"appearance"`
	BaseImagePrompt string `json:"base_image_prompt"`
}

type Agent1Node struct {
	NarrativeNodeID string                 `json:"narrative_node_id"`
	Title           string                 `json:"title"`
	PlotSummary     string                 `json:"plot_summary"`
	IsEntry         bool                   `json:"is_entry"`
	Choices         []NarrativeChoiceDraft `json:"choices"`
}

type Agent1Output struct {
	StartNarrativeNodeID string            `json:"start_narrative_node_id"`
	GlobalStoryline      string            `json:"global_storyline"`
	Characters           []Agent1Character `json:"characters"`
	GraphSkeleton        []Agent1Node      `json:"graph_skeleton"`
}

func (s *NarrativePackageService) RunAgent1Architect(userIdea string, drama models.Drama) (*Agent1Output, error) {
	idea := strings.TrimSpace(userIdea)
	if idea == "" {
		idea = strings.TrimSpace(drama.Title)
	}
	if idea == "" {
		return nil, fmt.Errorf("user_idea and drama title are empty")
	}

	system, err := narrativePromptFS.ReadFile("prompts/narrative/agent1_architect.md")
	if err != nil {
		return nil, fmt.Errorf("narrative agent1: read system prompt: %w", err)
	}

	tpl, err := template.New("agent1").Parse(string(system))
	if err != nil {
		return nil, fmt.Errorf("narrative agent1: parse template: %w", err)
	}

	var promptBuf bytes.Buffer
	_ = tpl.Execute(&promptBuf, struct {
		DramaTitle, UserIdea, Style, AspectRatio string
	}{
		DramaTitle:  drama.Title,
		UserIdea:    idea,
		Style:       drama.Style,
		AspectRatio: drama.AspectRatio,
	})

	userBlock := "Please execute Agent 1: The Architect based on the system instructions and return the JSON."

	_, err = s.ai.GetAIClient("text")
	if err != nil {
		return nil, fmt.Errorf("narrative agent1: no text AI client configured")
	}

	s.log.Infow("narrative agent1: starting global graph and character generation")

	raw, err := s.generateNarrativeText(userBlock, promptBuf.String(), 16384)
	if err != nil {
		return nil, fmt.Errorf("narrative agent1: text generation failed: %w", err)
	}

	raw = stripJSONFence(raw)
	var out Agent1Output
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, fmt.Errorf("narrative agent1: model output is not valid JSON: %w\nOutput was: %s", err, raw[:minInt(200, len(raw))])
	}

	if err := validateAgent1Graph(out); err != nil {
		return nil, fmt.Errorf("narrative agent1: invalid graph skeleton: %w", err)
	}

	return &out, nil
}

func validateAgent1Graph(out Agent1Output) error {
	if len(out.GraphSkeleton) < 15 || len(out.GraphSkeleton) > 20 {
		return fmt.Errorf("expected between 15 and 20 nodes, got %d", len(out.GraphSkeleton))
	}
	if strings.TrimSpace(out.StartNarrativeNodeID) == "" {
		return fmt.Errorf("start_narrative_node_id is required")
	}

	byID := make(map[string]Agent1Node, len(out.GraphSkeleton))
	entryCount := 0
	for _, node := range out.GraphSkeleton {
		id := strings.TrimSpace(node.NarrativeNodeID)
		if id == "" {
			return fmt.Errorf("node id is required")
		}
		if _, exists := byID[id]; exists {
			return fmt.Errorf("duplicate node id %q", id)
		}
		node.NarrativeNodeID = id
		byID[id] = node
		if node.IsEntry {
			entryCount++
		}
	}
	if _, ok := byID[out.StartNarrativeNodeID]; !ok {
		return fmt.Errorf("start node %q is not in graph", out.StartNarrativeNodeID)
	}
	if entryCount != 1 {
		return fmt.Errorf("expected exactly one entry node, got %d", entryCount)
	}

	branchingNodes := 0
	parents := make(map[string]int)
	for _, node := range out.GraphSkeleton {
		if len(node.Choices) > 1 {
			branchingNodes++
		}
		for _, choice := range node.Choices {
			next := strings.TrimSpace(choice.NextNarrativeNodeID)
			if next == "" {
				return fmt.Errorf("node %s has an empty choice target", node.NarrativeNodeID)
			}
			if _, ok := byID[next]; !ok {
				return fmt.Errorf("node %s targets missing node %s", node.NarrativeNodeID, next)
			}
			parents[next]++
		}
	}
	if branchingNodes == 0 {
		return fmt.Errorf("graph has no branching node")
	}
	if len(parents) < len(out.GraphSkeleton)-1 {
		return fmt.Errorf("graph is too disconnected")
	}

	seen := map[string]bool{out.StartNarrativeNodeID: true}
	queue := []string{out.StartNarrativeNodeID}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		for _, choice := range byID[current].Choices {
			next := choice.NextNarrativeNodeID
			if !seen[next] {
				seen[next] = true
				queue = append(queue, next)
			}
		}
	}
	if len(seen) != len(out.GraphSkeleton) {
		return fmt.Errorf("only %d of %d nodes are reachable from %s", len(seen), len(out.GraphSkeleton), out.StartNarrativeNodeID)
	}
	return nil
}

type Agent2Outfit struct {
	CharacterName string `json:"character_name"`
	OutfitName    string `json:"outfit_name"`
	OutfitPrompt  string `json:"outfit_prompt"`
}

type Agent2StateSnapshot struct {
	Timeline          string `json:"timeline"`
	CharacterStatuses string `json:"character_statuses"`
	KeyItemsLocations string `json:"key_items_locations"`
}

type Agent2Scene struct {
	LocationName string `json:"location_name"`
	ScenePrompt  string `json:"scene_prompt"`
}

type Agent2EpisodeData struct {
	NarrativeNodeID string              `json:"narrative_node_id"`
	MicroBeats      []string            `json:"micro_beats"`
	StateSnapshotT  Agent2StateSnapshot `json:"state_snapshot_t"`
	EpisodeOutfits  []Agent2Outfit      `json:"episode_outfits"`
	EpisodeScenes   []Agent2Scene       `json:"episode_scenes"`
}

type Agent2Output struct {
	EpisodesData []Agent2EpisodeData `json:"episodes_data"`
}

func (s *NarrativePackageService) RunAgent2Builder(dramaTitle string, globalCharacters []Agent1Character, graphSkeleton []Agent1Node) (*Agent2Output, error) {
	system, err := narrativePromptFS.ReadFile("prompts/narrative/agent2_builder.md")
	if err != nil {
		return nil, fmt.Errorf("narrative agent2: read system prompt: %w", err)
	}

	tpl, err := template.New("agent2").Parse(string(system))
	if err != nil {
		return nil, fmt.Errorf("narrative agent2: parse template: %w", err)
	}

	globalCharsJSONBytes, _ := json.MarshalIndent(globalCharacters, "", "  ")
	graphSkeletonJSONBytes, _ := json.MarshalIndent(graphSkeleton, "", "  ")

	var promptBuf bytes.Buffer
	_ = tpl.Execute(&promptBuf, struct {
		DramaTitle           string
		GlobalCharactersJSON string
		GraphSkeletonJSON    string
	}{
		DramaTitle:           dramaTitle,
		GlobalCharactersJSON: string(globalCharsJSONBytes),
		GraphSkeletonJSON:    string(graphSkeletonJSONBytes),
	})

	userBlock := "Please execute Agent 2: The Builder for ALL episodes and return the JSON array."

	_, err = s.ai.GetAIClient("text")
	if err != nil {
		return nil, fmt.Errorf("narrative agent2: no text AI client configured")
	}

	// Allow large output for 15 episodes
	raw, err := s.generateNarrativeText(userBlock, promptBuf.String(), 32768)
	if err != nil {
		return nil, fmt.Errorf("narrative agent2: text generation failed: %w", err)
	}

	raw = stripJSONFence(raw)
	var out Agent2Output
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, fmt.Errorf("narrative agent2: model output is not valid JSON: %w\nOutput was: %s", err, raw[:minInt(200, len(raw))])
	}

	if len(out.EpisodesData) != len(graphSkeleton) {
		s.log.Warnw("Agent 2 output length mismatch", "expected", len(graphSkeleton), "got", len(out.EpisodesData))
	}

	return &out, nil
}

func (s *NarrativePackageService) RunAgent2BuilderEpisode(dramaTitle string, globalCharacters []Agent1Character, graphSkeleton []Agent1Node, currentNode Agent1Node, incomingStates []Agent2StateSnapshot, priorEpisodeSummaries []string) (*Agent2EpisodeData, error) {
	system, err := narrativePromptFS.ReadFile("prompts/narrative/agent2_builder.md")
	if err != nil {
		return nil, fmt.Errorf("narrative agent2: read system prompt: %w", err)
	}

	tpl, err := template.New("agent2_episode").Parse(string(system))
	if err != nil {
		return nil, fmt.Errorf("narrative agent2: parse template: %w", err)
	}

	globalCharsJSONBytes, _ := json.MarshalIndent(globalCharacters, "", "  ")
	graphSkeletonJSONBytes, _ := json.MarshalIndent(graphSkeleton, "", "  ")
	currentNodeJSONBytes, _ := json.MarshalIndent(currentNode, "", "  ")
	incomingStatesJSONBytes, _ := json.MarshalIndent(incomingStates, "", "  ")
	priorContextJSONBytes, _ := json.MarshalIndent(priorEpisodeSummaries, "", "  ")

	var promptBuf bytes.Buffer
	_ = tpl.Execute(&promptBuf, struct {
		DramaTitle                 string
		GlobalCharactersJSON       string
		GraphSkeletonJSON          string
		CurrentNodeJSON            string
		IncomingStateSnapshotsJSON string
		PriorEpisodeSummariesJSON  string
	}{
		DramaTitle:                 dramaTitle,
		GlobalCharactersJSON:       string(globalCharsJSONBytes),
		GraphSkeletonJSON:          string(graphSkeletonJSONBytes),
		CurrentNodeJSON:            string(currentNodeJSONBytes),
		IncomingStateSnapshotsJSON: string(incomingStatesJSONBytes),
		PriorEpisodeSummariesJSON:  string(priorContextJSONBytes),
	})

	userBlock := "Please execute Agent 2: The Builder for the CURRENT episode only and return one JSON object."
	_, err = s.ai.GetAIClient("text")
	if err != nil {
		return nil, fmt.Errorf("narrative agent2: no text AI client configured")
	}

	raw, err := s.generateNarrativeText(userBlock, promptBuf.String(), 8192)
	if err != nil {
		return nil, fmt.Errorf("narrative agent2: text generation failed: %w", err)
	}

	raw = stripJSONFence(raw)
	var data Agent2EpisodeData
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		var wrapper struct {
			EpisodeData Agent2EpisodeData `json:"episode_data"`
		}
		if wrapErr := json.Unmarshal([]byte(raw), &wrapper); wrapErr != nil {
			return nil, fmt.Errorf("narrative agent2: model output is not valid JSON: %w\nOutput was: %s", err, raw[:minInt(200, len(raw))])
		}
		data = wrapper.EpisodeData
	}
	if data.NarrativeNodeID != currentNode.NarrativeNodeID {
		return nil, fmt.Errorf("narrative agent2: expected node %s, got %s", currentNode.NarrativeNodeID, data.NarrativeNodeID)
	}
	return &data, nil
}

type Agent3ScriptData struct {
	NarrativeNodeID string `json:"narrative_node_id"`
	ScriptContent   string `json:"script_content"`
}

type Agent3Output struct {
	ScriptsData []Agent3ScriptData `json:"scripts_data"`
}

func (s *NarrativePackageService) RunAgent3Designer(dramaTitle string, globalCharacters []Agent1Character, episodesData []Agent2EpisodeData) (*Agent3Output, error) {
	system, err := narrativePromptFS.ReadFile("prompts/narrative/agent3_designer.md")
	if err != nil {
		return nil, fmt.Errorf("narrative agent3: read system prompt: %w", err)
	}

	tpl, err := template.New("agent3").Parse(string(system))
	if err != nil {
		return nil, fmt.Errorf("narrative agent3: parse template: %w", err)
	}

	globalCharsJSONBytes, _ := json.MarshalIndent(globalCharacters, "", "  ")
	episodesDataJSONBytes, _ := json.MarshalIndent(episodesData, "", "  ")

	var promptBuf bytes.Buffer
	_ = tpl.Execute(&promptBuf, struct {
		DramaTitle           string
		GlobalCharactersJSON string
		EpisodesDataJSON     string
	}{
		DramaTitle:           dramaTitle,
		GlobalCharactersJSON: string(globalCharsJSONBytes),
		EpisodesDataJSON:     string(episodesDataJSONBytes),
	})

	userBlock := "Please execute Agent 3: The Designer to write the markdown scripts for ALL episodes and return the JSON array."

	_, err = s.ai.GetAIClient("text")
	if err != nil {
		return nil, fmt.Errorf("narrative agent3: no text AI client configured")
	}

	// Allow large output for 15 episodes worth of scripts
	raw, err := s.generateNarrativeText(userBlock, promptBuf.String(), 65536)
	if err != nil {
		return nil, fmt.Errorf("narrative agent3: text generation failed: %w", err)
	}

	raw = stripJSONFence(raw)
	var out Agent3Output
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, fmt.Errorf("narrative agent3: model output is not valid JSON: %w\nOutput was: %s", err, raw[:minInt(200, len(raw))])
	}

	if len(out.ScriptsData) != len(episodesData) {
		s.log.Warnw("Agent 3 output length mismatch", "expected", len(episodesData), "got", len(out.ScriptsData))
	}

	return &out, nil
}

func (s *NarrativePackageService) RunAgent3DesignerEpisode(dramaTitle string, globalCharacters []Agent1Character, graphNode Agent1Node, episodeData Agent2EpisodeData) (*Agent3ScriptData, error) {
	system, err := narrativePromptFS.ReadFile("prompts/narrative/agent3_designer.md")
	if err != nil {
		return nil, fmt.Errorf("narrative agent3: read system prompt: %w", err)
	}

	tpl, err := template.New("agent3_episode").Parse(string(system))
	if err != nil {
		return nil, fmt.Errorf("narrative agent3: parse template: %w", err)
	}

	globalCharsJSONBytes, _ := json.MarshalIndent(globalCharacters, "", "  ")
	graphNodeJSONBytes, _ := json.MarshalIndent(graphNode, "", "  ")
	episodeDataJSONBytes, _ := json.MarshalIndent(episodeData, "", "  ")

	var promptBuf bytes.Buffer
	_ = tpl.Execute(&promptBuf, struct {
		DramaTitle           string
		GlobalCharactersJSON string
		GraphNodeJSON        string
		EpisodeDataJSON      string
		EpisodesDataJSON     string
	}{
		DramaTitle:           dramaTitle,
		GlobalCharactersJSON: string(globalCharsJSONBytes),
		GraphNodeJSON:        string(graphNodeJSONBytes),
		EpisodeDataJSON:      string(episodeDataJSONBytes),
		EpisodesDataJSON:     string(episodeDataJSONBytes),
	})

	userBlock := "Please execute Agent 3: The Designer for the CURRENT episode only and return one JSON object."
	_, err = s.ai.GetAIClient("text")
	if err != nil {
		return nil, fmt.Errorf("narrative agent3: no text AI client configured")
	}

	raw, err := s.generateNarrativeText(userBlock, promptBuf.String(), 12288)
	if err != nil {
		return nil, fmt.Errorf("narrative agent3: text generation failed: %w", err)
	}

	raw = stripJSONFence(raw)
	var data Agent3ScriptData
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		var wrapper struct {
			ScriptData Agent3ScriptData `json:"script_data"`
		}
		if wrapErr := json.Unmarshal([]byte(raw), &wrapper); wrapErr != nil {
			return nil, fmt.Errorf("narrative agent3: model output is not valid JSON: %w\nOutput was: %s", err, raw[:minInt(200, len(raw))])
		}
		data = wrapper.ScriptData
	}
	if data.NarrativeNodeID != graphNode.NarrativeNodeID {
		return nil, fmt.Errorf("narrative agent3: expected node %s, got %s", graphNode.NarrativeNodeID, data.NarrativeNodeID)
	}
	data.ScriptContent = stripUIUXBlock(data.ScriptContent)
	return &data, nil
}

var jsonFenceRE = regexp.MustCompile("(?s)```(?:json)?\\s*([\\s\\S]*?)```")

func stripJSONFence(text string) string {
	text = strings.TrimSpace(text)
	if m := jsonFenceRE.FindStringSubmatch(text); len(m) > 1 {
		return strings.TrimSpace(m[1])
	}
	return text
}

func normalizeNarrativeGraph(pkg *NarrativeDramaPackage) *NarrativeDramaPackage {
	if pkg == nil || len(pkg.Episodes) == 0 {
		return nil
	}
	byID := make(map[string]NarrativeEpisodeDraft)
	for i := range pkg.Episodes {
		ep := pkg.Episodes[i]
		k := strings.TrimSpace(ep.NarrativeNodeID)
		if k == "" {
			continue
		}
		ep.NarrativeNodeID = k
		byID[k] = ep
	}
	start := strings.TrimSpace(pkg.StartNarrativeNodeID)
	if start == "" {
		start = "N101"
	}
	if _, ok := byID[start]; !ok {
		return nil
	}
	for _, ep := range byID {
		for _, c := range ep.Choices {
			t := strings.TrimSpace(c.NextNarrativeNodeID)
			if t != "" {
				if _, ok := byID[t]; !ok {
					return nil
				}
			}
		}
	}
	ordered := make([]NarrativeEpisodeDraft, 0, len(byID))
	seen := make(map[string]bool)
	queue := []string{start}
	for len(queue) > 0 {
		nid := queue[0]
		queue = queue[1:]
		if seen[nid] {
			continue
		}
		seen[nid] = true
		ep := byID[nid]
		ordered = append(ordered, ep)
		for _, c := range ep.Choices {
			t := strings.TrimSpace(c.NextNarrativeNodeID)
			if t != "" && !seen[t] {
				if _, ok := byID[t]; ok {
					queue = append(queue, t)
				}
			}
		}
	}
	for k, ep := range byID {
		if !seen[k] {
			ordered = append(ordered, ep)
		}
	}
	for i := range ordered {
		ordered[i].EpisodeNumber = i + 1
		ordered[i].IsEntry = ordered[i].NarrativeNodeID == start
	}
	return &NarrativeDramaPackage{
		StartNarrativeNodeID: start,
		Episodes:             ordered,
	}
}

// BuildStubNarrativeDramaPackage returns a small DAG: 1 fork (≤3 tracks) → merge → 2 endings (7 nodes N101…N107).
func BuildStubNarrativeDramaPackage(userIdea, dramaTitle string) *NarrativeDramaPackage {
	idea := strings.TrimSpace(userIdea)
	if idea == "" {
		idea = dramaTitle
	}
	title := dramaTitle
	if title == "" {
		title = "Untitled"
	}
	snip := idea
	if len([]rune(snip)) > 36 {
		snip = string([]rune(snip)[:35]) + "…"
	}

	script := func(nid, blurb string, extra string) string {
		return fmt.Sprintf(`# %s — %s

[SCENE: %s]
*( premise: %q · %s )*

[ACTION]
Hold the premise “%s”; parallel branches later merge—avoid an exploding number of endings.

[DIALOGUE]
**Narrator**
%s
%s
`, nid, title, blurb, idea, snip, snip, blurb, extra)
	}

	drafts := []NarrativeEpisodeDraft{
		{
			NarrativeNodeID: "N101", EpisodeNumber: 1,
			Title: fmt.Sprintf("Open · %s", snip), IsEntry: true,
			ScriptContent: script("N101", "Fork: at most three parallel tracks.", ""),
			Choices: []NarrativeChoiceDraft{
				{Label: "Branch 1 — cautious play", NextNarrativeNodeID: "N102"},
				{Label: "Branch 2 — direct confrontation", NextNarrativeNodeID: "N103"},
				{Label: "Branch 3 — temporary exit", NextNarrativeNodeID: "N104"},
			},
		},
		{
			NarrativeNodeID: "N102", EpisodeNumber: 2,
			Title:         "Track 1",
			ScriptContent: script("N102", "Cautious branch: time has a price.", "Heading to the shared merge node."),
			Choices:       []NarrativeChoiceDraft{{Label: "Merge", NextNarrativeNodeID: "N105"}},
		},
		{
			NarrativeNodeID: "N103", EpisodeNumber: 3,
			Title:         "Track 2",
			ScriptContent: script("N103", "Confrontation branch: tense but goal-clear.", "Same destination as the other branches."),
			Choices:       []NarrativeChoiceDraft{{Label: "Merge", NextNarrativeNodeID: "N105"}},
		},
		{
			NarrativeNodeID: "N104", EpisodeNumber: 4,
			Title:         "Track 3",
			ScriptContent: script("N104", "Escape branch: short-term sacrifice.", "Rejoin the main thread at merge."),
			Choices:       []NarrativeChoiceDraft{{Label: "Merge", NextNarrativeNodeID: "N105"}},
		},
		{
			NarrativeNodeID: "N105", EpisodeNumber: 5,
			Title:         "Merge",
			ScriptContent: script("N105", "After merge: only the final fork (few endings).", ""),
			Choices: []NarrativeChoiceDraft{
				{Label: "Ending tilt — justice / repair", NextNarrativeNodeID: "N106"},
				{Label: "Ending tilt — personal cost", NextNarrativeNodeID: "N107"},
			},
		},
		{
			NarrativeNodeID: "N106", EpisodeNumber: 6,
			Title:         "Ending A",
			ScriptContent: script("N106", "First ending — one closed path.", "**End** — this branch closes."),
			Choices:       []NarrativeChoiceDraft{},
		},
		{
			NarrativeNodeID: "N107", EpisodeNumber: 7,
			Title:         "Ending B",
			ScriptContent: script("N107", "Second ending — different tone, same premise.", "**End** — this branch closes."),
			Choices:       []NarrativeChoiceDraft{},
		},
	}
	return &NarrativeDramaPackage{
		StartNarrativeNodeID: "N101",
		Episodes:             drafts,
	}
}
