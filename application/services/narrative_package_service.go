package services

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"regexp"
	"strings"

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
Bám ý tưởng “%s”; nhánh song song sau đó hội tụ, tránh bùng nổ số kết.

[DIALOGUE]
**Dẫn**
%s
%s
`, nid, title, blurb, idea, snip, snip, blurb, extra)
	}

	drafts := []NarrativeEpisodeDraft{
		{
			NarrativeNodeID: "N101", EpisodeNumber: 1,
			Title: fmt.Sprintf("Mở · %s", snip), IsEntry: true,
			ScriptContent: script("N101", "Điểm rẽ: tối đa ba hướng xử lý song song.", ""),
			Choices: []NarrativeChoiceDraft{
				{Label: "Nhánh 1 — xử lý thận trọng", NextNarrativeNodeID: "N102"},
				{Label: "Nhánh 2 — đối đầu trực diện", NextNarrativeNodeID: "N103"},
				{Label: "Nhánh 3 — thoát hiểm tạm thời", NextNarrativeNodeID: "N104"},
			},
		},
		{
			NarrativeNodeID: "N102", EpisodeNumber: 2,
			Title:         "Track 1",
			ScriptContent: script("N102", "Nhánh thận trọng: thời gian đổi giá.", "Hướng tới nút hội tụ chung."),
			Choices:       []NarrativeChoiceDraft{{Label: "Hội tụ", NextNarrativeNodeID: "N105"}},
		},
		{
			NarrativeNodeID: "N103", EpisodeNumber: 3,
			Title:         "Track 2",
			ScriptContent: script("N103", "Nhánh đối đầu: căng nhưng rõ mục tiêu.", "Cùng đích với các nhánh khác."),
			Choices:       []NarrativeChoiceDraft{{Label: "Hội tụ", NextNarrativeNodeID: "N105"}},
		},
		{
			NarrativeNodeID: "N104", EpisodeNumber: 4,
			Title:         "Track 3",
			ScriptContent: script("N104", "Nhánh thoát hiểm: hy sinh ngắn hạn.", "Gặp lại mạch chính ở node merge."),
			Choices:       []NarrativeChoiceDraft{{Label: "Hội tụ", NextNarrativeNodeID: "N105"}},
		},
		{
			NarrativeNodeID: "N105", EpisodeNumber: 5,
			Title:         "Hội tụ",
			ScriptContent: script("N105", "Sau merge: chỉ còn phân nhánh cuối (tối đa vài kết).", ""),
			Choices: []NarrativeChoiceDraft{
				{Label: "Kết thiên về công lý / sửa chữa", NextNarrativeNodeID: "N106"},
				{Label: "Kết thiên về cái giá cá nhân", NextNarrativeNodeID: "N107"},
			},
		},
		{
			NarrativeNodeID: "N106", EpisodeNumber: 6,
			Title:         "Kết A",
			ScriptContent: script("N106", "Điểm kết 1 — đóng một đường duy nhất.", "**Kết** — nhánh này khép."),
			Choices:       []NarrativeChoiceDraft{},
		},
		{
			NarrativeNodeID: "N107", EpisodeNumber: 7,
			Title:         "Kết B",
			ScriptContent: script("N107", "Điểm kết 2 — tone khác, cùng premise.", "**Kết** — nhánh này khép."),
			Choices:       []NarrativeChoiceDraft{},
		},
	}
	return &NarrativeDramaPackage{
		StartNarrativeNodeID: "N101",
		Episodes:             drafts,
	}
}
