package services

import "testing"

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
