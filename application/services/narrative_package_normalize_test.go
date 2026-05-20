package services

import (
	"fmt"
	"testing"
)

func TestNormalizeNarrativeGraph_BFS(t *testing.T) {
	pkg := &NarrativeDramaPackage{
		StartNarrativeNodeID: "N101",
		Episodes: []NarrativeEpisodeDraft{
			{NarrativeNodeID: "N101", EpisodeNumber: 99, Choices: []NarrativeChoiceDraft{{Label: "a", NextNarrativeNodeID: "N102"}}},
			{NarrativeNodeID: "N102", EpisodeNumber: 1, Choices: []NarrativeChoiceDraft{}},
		},
	}
	out := normalizeNarrativeGraph(pkg)
	if out == nil || len(out.Episodes) != 2 {
		t.Fatalf("expected 2 episodes, got %v", out)
	}
	if out.Episodes[0].NarrativeNodeID != "N101" || out.Episodes[0].EpisodeNumber != 1 {
		t.Fatalf("first should be N101 epnum 1: %+v", out.Episodes[0])
	}
	if !out.Episodes[0].IsEntry || out.Episodes[1].IsEntry {
		t.Fatalf("entry flags wrong: %+v %+v", out.Episodes[0], out.Episodes[1])
	}
	if out.Episodes[1].EpisodeNumber != 2 {
		t.Fatalf("second epnum: %+v", out.Episodes[1])
	}
}

func TestNormalizeNarrativeGraph_BrokenEdge(t *testing.T) {
	pkg := &NarrativeDramaPackage{
		StartNarrativeNodeID: "N101",
		Episodes: []NarrativeEpisodeDraft{
			{NarrativeNodeID: "N101", Choices: []NarrativeChoiceDraft{{NextNarrativeNodeID: "N999"}}},
		},
	}
	if normalizeNarrativeGraph(pkg) != nil {
		t.Fatal("expected nil for broken edge")
	}
}

// Convergence: three parents → same merge id (BFS visits merge once after parallel tracks).
func TestNormalizeNarrativeGraph_MergeConvergence(t *testing.T) {
	pkg := &NarrativeDramaPackage{
		StartNarrativeNodeID: "N101",
		Episodes: []NarrativeEpisodeDraft{
			{NarrativeNodeID: "N101", Choices: []NarrativeChoiceDraft{
				{NextNarrativeNodeID: "N102"}, {NextNarrativeNodeID: "N103"}, {NextNarrativeNodeID: "N104"},
			}},
			{NarrativeNodeID: "N102", Choices: []NarrativeChoiceDraft{{NextNarrativeNodeID: "N105"}}},
			{NarrativeNodeID: "N103", Choices: []NarrativeChoiceDraft{{NextNarrativeNodeID: "N105"}}},
			{NarrativeNodeID: "N104", Choices: []NarrativeChoiceDraft{{NextNarrativeNodeID: "N105"}}},
			{NarrativeNodeID: "N105", Choices: []NarrativeChoiceDraft{
				{NextNarrativeNodeID: "N106"}, {NextNarrativeNodeID: "N107"},
			}},
			{NarrativeNodeID: "N106", Choices: []NarrativeChoiceDraft{}},
			{NarrativeNodeID: "N107", Choices: []NarrativeChoiceDraft{}},
		},
	}
	out := normalizeNarrativeGraph(pkg)
	if out == nil || len(out.Episodes) != 7 {
		t.Fatalf("expected 7 episodes, got %v", out)
	}
	want := []string{"N101", "N102", "N103", "N104", "N105", "N106", "N107"}
	for i, id := range want {
		if out.Episodes[i].NarrativeNodeID != id || out.Episodes[i].EpisodeNumber != i+1 {
			t.Fatalf("idx %d: want %s ep#%d, got %+v", i, id, i+1, out.Episodes[i])
		}
	}
}

func TestBuildStubNarrativeDramaPackage_Shape(t *testing.T) {
	pkg := BuildStubNarrativeDramaPackage("idea", "T")
	if len(pkg.Episodes) != 7 {
		t.Fatalf("stub should have 7 nodes, got %d", len(pkg.Episodes))
	}
	if len(pkg.Episodes[0].Choices) != 3 {
		t.Fatalf("entry should fork to 3 tracks, got %d", len(pkg.Episodes[0].Choices))
	}
	ends := 0
	for _, ep := range pkg.Episodes {
		if len(ep.Choices) == 0 {
			ends++
		}
	}
	if ends != 2 {
		t.Fatalf("expected 2 endings, got %d", ends)
	}
}

func TestValidateAgent1Graph_ValidFifteenNodeBranchingGraph(t *testing.T) {
	out := Agent1Output{
		StartNarrativeNodeID: "N101",
		GraphSkeleton: []Agent1Node{
			{NarrativeNodeID: "N101", IsEntry: true, Choices: []NarrativeChoiceDraft{{NextNarrativeNodeID: "N102"}, {NextNarrativeNodeID: "N103"}}},
			{NarrativeNodeID: "N102", Choices: []NarrativeChoiceDraft{{NextNarrativeNodeID: "N104"}}},
			{NarrativeNodeID: "N103", Choices: []NarrativeChoiceDraft{{NextNarrativeNodeID: "N105"}}},
			{NarrativeNodeID: "N104", Choices: []NarrativeChoiceDraft{{NextNarrativeNodeID: "N106"}}},
			{NarrativeNodeID: "N105", Choices: []NarrativeChoiceDraft{{NextNarrativeNodeID: "N107"}}},
			{NarrativeNodeID: "N106", Choices: []NarrativeChoiceDraft{{NextNarrativeNodeID: "N108"}}},
			{NarrativeNodeID: "N107", Choices: []NarrativeChoiceDraft{{NextNarrativeNodeID: "N109"}}},
			{NarrativeNodeID: "N108", Choices: []NarrativeChoiceDraft{{NextNarrativeNodeID: "N110"}}},
			{NarrativeNodeID: "N109", Choices: []NarrativeChoiceDraft{{NextNarrativeNodeID: "N111"}}},
			{NarrativeNodeID: "N110", Choices: []NarrativeChoiceDraft{{NextNarrativeNodeID: "N112"}}},
			{NarrativeNodeID: "N111", Choices: []NarrativeChoiceDraft{{NextNarrativeNodeID: "N113"}}},
			{NarrativeNodeID: "N112", Choices: []NarrativeChoiceDraft{{NextNarrativeNodeID: "N114"}}},
			{NarrativeNodeID: "N113", Choices: []NarrativeChoiceDraft{{NextNarrativeNodeID: "N115"}}},
			{NarrativeNodeID: "N114"},
			{NarrativeNodeID: "N115"},
		},
	}
	if err := validateAgent1Graph(out); err != nil {
		t.Fatalf("expected valid graph: %v", err)
	}
}

func TestValidateAgent1Graph_RejectsBrokenEdge(t *testing.T) {
	out := Agent1Output{
		StartNarrativeNodeID: "N101",
		GraphSkeleton:        make([]Agent1Node, 15),
	}
	for i := range out.GraphSkeleton {
		out.GraphSkeleton[i] = Agent1Node{NarrativeNodeID: fmt.Sprintf("N%03d", 101+i)}
	}
	out.GraphSkeleton[0] = Agent1Node{NarrativeNodeID: "N101", IsEntry: true, Choices: []NarrativeChoiceDraft{{NextNarrativeNodeID: "N999"}}}
	if err := validateAgent1Graph(out); err == nil {
		t.Fatal("expected broken edge to be rejected")
	}
}

func TestValidateAgent1Graph_RejectsLinearOnlyGraph(t *testing.T) {
	nodes := make([]Agent1Node, 15)
	for i := range nodes {
		id := 101 + i
		nodes[i] = Agent1Node{NarrativeNodeID: fmt.Sprintf("N%d", id)}
		if i < len(nodes)-1 {
			nodes[i].Choices = []NarrativeChoiceDraft{{NextNarrativeNodeID: fmt.Sprintf("N%d", id+1)}}
		}
	}
	nodes[0].NarrativeNodeID = "N101"
	nodes[0].IsEntry = true
	for i := 1; i < len(nodes); i++ {
		nodes[i].NarrativeNodeID = fmt.Sprintf("N%d", 101+i)
	}
	out := Agent1Output{StartNarrativeNodeID: "N101", GraphSkeleton: nodes}
	if err := validateAgent1Graph(out); err == nil {
		t.Fatal("expected linear-only graph to be rejected")
	}
}

func TestGenerateNarrativeEpisodesRejectsInvalidAgentStepBeforeDB(t *testing.T) {
	_, err := (&DramaService{}).GenerateNarrativeEpisodes("1", NarrativeGenerateRequest{AgentStep: 4})
	if err == nil {
		t.Fatal("expected invalid agent step error")
	}
}

func TestNormalizeStoryboardNarrationMutualExclusion(t *testing.T) {
	sb := Storyboard{Dialogue: `A: "Go."`, Narration: "He orders everyone forward."}
	normalizeStoryboardNarration(&sb)
	if sb.Narration != "" {
		t.Fatalf("dialogue should clear narration, got %q", sb.Narration)
	}

	sb = Storyboard{Narration: "  The secret finally surfaces.  "}
	normalizeStoryboardNarration(&sb)
	if sb.Narration != "The secret finally surfaces." {
		t.Fatalf("expected trimmed narration, got %q", sb.Narration)
	}
}

func TestIsTransientModelError(t *testing.T) {
	transient := fmt.Errorf(`API error (status 503): {"error":{"status":"UNAVAILABLE","message":"high demand"}}`)
	if !isTransientModelError(transient) {
		t.Fatal("expected 503 unavailable high-demand error to be transient")
	}
	permanent := fmt.Errorf("API error: invalid api key")
	if isTransientModelError(permanent) {
		t.Fatal("did not expect invalid api key to be transient")
	}
}
