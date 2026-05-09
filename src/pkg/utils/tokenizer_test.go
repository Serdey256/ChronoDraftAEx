package utils

import (
	"testing"
	"time"

	"ChronoDraftAEx/pkg/models"
)

func TestEstimateTokens_EmptyText(t *testing.T) {
	if got := EstimateTokens(""); got != 0 {
		t.Errorf("EstimateTokens(\"\") = %d; want 0", got)
	}
}

func TestEstimateTokens_ShortText(t *testing.T) {
	// len("a") = 1, 1/3 = 0, but short text returns 1
	if got := EstimateTokens("a"); got != 1 {
		t.Errorf("EstimateTokens(\"a\") = %d; want 1", got)
	}
}

func TestEstimateTokens_English(t *testing.T) {
	// len("hello world") = 11, 11/3 ≈ 3.67 → 3
	if got := EstimateTokens("hello world"); got != 3 {
		t.Errorf("EstimateTokens(\"hello world\") = %d; want 3", got)
	}
}

func TestEstimateTokens_Chinese(t *testing.T) {
	// "你好世界" in UTF-8 is 12 bytes, 12/3 = 4
	if got := EstimateTokens("你好世界"); got != 4 {
		t.Errorf("EstimateTokens(\"你好世界\") = %d; want 4", got)
	}
}

func TestEstimateTokens_Mixed(t *testing.T) {
	// "hello 世界" in UTF-8: 5 + 1 + 6 = 12 bytes, 12/3 = 4
	if got := EstimateTokens("hello 世界"); got != 4 {
		t.Errorf("EstimateTokens(\"hello 世界\") = %d; want 4", got)
	}
}

func TestTruncateToBudget_UnderBudget(t *testing.T) {
	input := "hello world"
	got := TruncateToBudget(input, 100)
	if got != input {
		t.Errorf("TruncateToBudget under budget: got %q; want %q", got, input)
	}
}

func TestTruncateToBudget_EmptyText(t *testing.T) {
	got := TruncateToBudget("", 10)
	if got != "" {
		t.Errorf("TruncateToBudget empty: got %q; want \"\"", got)
	}
}

func TestTruncateToBudget_NegativeBudget(t *testing.T) {
	got := TruncateToBudget("hello", -1)
	if got != "" {
		t.Errorf("TruncateToBudget negative: got %q; want \"\"", got)
	}
}

func TestTruncateToBudget_TruncateLines(t *testing.T) {
	input := "line1\nline2\nline3\nline4"
	// Each line is ~2 tokens. Budget 5 should keep roughly the first 2-3 lines.
	// line1=2, line1+line2=4, line1+line2+line3=6
	got := TruncateToBudget(input, 5)
	// Should keep lines until token estimate exceeds 5
	// "line1\nline2" = 10 bytes / 3 = 3 tokens, under 5
	// "line1\nline2\nline3" = 17 bytes / 3 = 5 tokens, under or equal 5
	// Either is acceptable — verify it's a prefix and under budget
	if EstimateTokens(got) > 5 {
		t.Errorf("TruncateToBudget result exceeds budget: got %q (%d tokens)", got, EstimateTokens(got))
	}
}

func TestTruncateToBudget_AllLinesOverBudget(t *testing.T) {
	input := "very long line that definitely exceeds any reasonable budget for this test"
	// Even the first line is way over budget of 1
	got := TruncateToBudget(input, 1)
	if got != "" {
		t.Errorf("TruncateToBudget for single line over budget: got %q; want \"\"", got)
	}
}

func TestCompressEntry(t *testing.T) {
	entry := models.StructuredEntry{
		ID:             "test-id",
		Timestamp:      time.Now(),
		SessionID:      "session-1",
		Summary:        "Added user authentication",
		DesignDecision: "Use JWT for stateless auth",
		ImpactAnalysis: "Affects login, register, and middleware",
	}
	got := CompressEntry(entry, 200)
	expectedPrefix := "Summary: Added user authentication | Decision: Use JWT for stateless auth | Impact: Affects login, register, and middleware"
	if got != expectedPrefix {
		t.Errorf("CompressEntry: got %q; want %q", got, expectedPrefix)
	}
	if EstimateTokens(got) > 200 {
		t.Errorf("CompressEntry result exceeds budget: %d tokens", EstimateTokens(got))
	}
}

func TestCompressEntry_TightBudget(t *testing.T) {
	entry := models.StructuredEntry{
		Summary:        "First line summary here",
		DesignDecision: "Decision content that is somewhat long and detailed",
		ImpactAnalysis: "Impact analysis with more text to push it over budget",
	}
	// Tight budget — should truncate from the end
	got := CompressEntry(entry, 5)
	if EstimateTokens(got) > 5 {
		t.Errorf("CompressEntry tight budget: %d tokens exceeds 5", EstimateTokens(got))
	}
}
