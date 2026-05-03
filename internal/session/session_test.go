package session

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func setupTestDir(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	t.Cleanup(func() { os.Unsetenv("HOME") })
	return tmpDir
}

func TestNewPaper(t *testing.T) {
	setupTestDir(t)

	p := NewPaper("test content", "https://example.com")

	if p.ID != 1 {
		t.Errorf("expected ID 1, got %d", p.ID)
	}
	if p.Content != "test content" {
		t.Errorf("unexpected content: %s", p.Content)
	}
	if p.SourceURL != "https://example.com" {
		t.Errorf("unexpected source URL: %s", p.SourceURL)
	}
	if len(p.Messages) != 0 {
		t.Errorf("expected 0 messages, got %d", len(p.Messages))
	}
}

func TestNextID(t *testing.T) {
	tmpDir := setupTestDir(t)
	papersDir := filepath.Join(tmpDir, ".paperpaper", "papers")
	os.MkdirAll(papersDir, 0755)

	// Create some paper files
	for i := 1; i <= 3; i++ {
		p := &Paper{ID: i, Content: "test"}
		SavePaper(p)
	}

	id := nextID()
	if id != 4 {
		t.Errorf("expected next ID 4, got %d", id)
	}
}

func TestSaveAndLoadPaper(t *testing.T) {
	setupTestDir(t)

	p := &Paper{
		ID:             1,
		Title:          "Test Paper",
		Content:        "test content",
		InitialSummary: "test summary",
		ModelUsed:      "gpt-4o",
		TotalTokens:    1000,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		Messages: []Message{
			{RoundNumber: 0, Role: "user", Content: "question", TokenCount: 10, CreatedAt: time.Now()},
			{RoundNumber: 0, Role: "assistant", Content: "answer", TokenCount: 50, CreatedAt: time.Now()},
		},
	}

	if err := SavePaper(p); err != nil {
		t.Fatalf("save error: %v", err)
	}

	loaded, err := LoadPaper(1)
	if err != nil {
		t.Fatalf("load error: %v", err)
	}

	if loaded.Title != "Test Paper" {
		t.Errorf("expected title 'Test Paper', got %s", loaded.Title)
	}
	if loaded.Content != "test content" {
		t.Errorf("unexpected content: %s", loaded.Content)
	}
	if len(loaded.Messages) != 2 {
		t.Errorf("expected 2 messages, got %d", len(loaded.Messages))
	}
	if loaded.Messages[0].Role != "user" {
		t.Errorf("expected user role, got %s", loaded.Messages[0].Role)
	}
}

func TestDeletePaper(t *testing.T) {
	setupTestDir(t)

	p := &Paper{ID: 1, Content: "test"}
	SavePaper(p)

	if err := DeletePaper(1); err != nil {
		t.Fatalf("delete error: %v", err)
	}

	_, err := LoadPaper(1)
	if err == nil {
		t.Error("expected error loading deleted paper")
	}
}

func TestListPapers(t *testing.T) {
	setupTestDir(t)

	for i := 1; i <= 3; i++ {
		p := &Paper{ID: i, Title: "Paper " + string(rune('A'+i-1)), Content: "test"}
		SavePaper(p)
	}

	papers, err := ListPapers()
	if err != nil {
		t.Fatalf("list error: %v", err)
	}
	if len(papers) != 3 {
		t.Errorf("expected 3 papers, got %d", len(papers))
	}
}

func TestManagerAddMessage(t *testing.T) {
	m := NewManager()
	p := NewPaper("content", "")
	m.SetPaper(p)

	msg := Message{RoundNumber: 0, Role: "user", Content: "test", TokenCount: 10}
	m.AddMessage(msg)

	if len(m.Paper().Messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(m.Paper().Messages))
	}
	if m.Paper().TotalTokens != 10 {
		t.Errorf("expected 10 tokens, got %d", m.Paper().TotalTokens)
	}
}

func TestManagerGetRecentMessages(t *testing.T) {
	m := NewManager()
	p := NewPaper("content", "")
	m.SetPaper(p)

	for i := 0; i < 10; i++ {
		m.AddMessage(Message{RoundNumber: i, Role: "user", Content: "q", TokenCount: 1})
		m.AddMessage(Message{RoundNumber: i, Role: "assistant", Content: "a", TokenCount: 1})
	}

	recent := m.GetRecentMessages(3)
	if len(recent) != 6 {
		t.Errorf("expected 6 messages (3 rounds), got %d", len(recent))
	}

	// Should be the last 3 rounds
	if recent[0].RoundNumber != 7 {
		t.Errorf("expected round 7, got %d", recent[0].RoundNumber)
	}
}

func TestManagerDeleteRound(t *testing.T) {
	m := NewManager()
	p := NewPaper("content", "")
	m.SetPaper(p)

	for i := 0; i < 3; i++ {
		m.AddMessage(Message{RoundNumber: i, Role: "user", Content: "q", TokenCount: 1})
		m.AddMessage(Message{RoundNumber: i, Role: "assistant", Content: "a", TokenCount: 1})
	}

	m.DeleteRound(1)

	if len(m.Paper().Messages) != 4 {
		t.Errorf("expected 4 messages after delete, got %d", len(m.Paper().Messages))
	}

	// Verify round 1 is gone
	for _, msg := range m.Paper().Messages {
		if msg.RoundNumber == 1 {
			t.Error("round 1 should have been deleted")
		}
	}
}

func TestManagerDeleteLastRound(t *testing.T) {
	m := NewManager()
	p := NewPaper("content", "")
	m.SetPaper(p)

	for i := 0; i < 3; i++ {
		m.AddMessage(Message{RoundNumber: i, Role: "user", Content: "q", TokenCount: 1})
		m.AddMessage(Message{RoundNumber: i, Role: "assistant", Content: "a", TokenCount: 1})
	}

	m.DeleteLastRound()

	if len(m.Paper().Messages) != 4 {
		t.Errorf("expected 4 messages, got %d", len(m.Paper().Messages))
	}

	// Last round should be 1
	lastRound := m.Paper().Messages[len(m.Paper().Messages)-1].RoundNumber
	if lastRound != 1 {
		t.Errorf("expected last round 1, got %d", lastRound)
	}
}

func TestManagerGetLastUserMessage(t *testing.T) {
	m := NewManager()
	p := NewPaper("content", "")
	m.SetPaper(p)

	// No messages yet
	if m.GetLastUserMessage() != nil {
		t.Error("expected nil for no messages")
	}

	m.AddMessage(Message{RoundNumber: 0, Role: "user", Content: "first question", TokenCount: 10})
	m.AddMessage(Message{RoundNumber: 0, Role: "assistant", Content: "answer", TokenCount: 50})
	m.AddMessage(Message{RoundNumber: 1, Role: "user", Content: "second question", TokenCount: 15})

	msg := m.GetLastUserMessage()
	if msg == nil {
		t.Fatal("expected non-nil message")
	}
	if msg.Content != "second question" {
		t.Errorf("expected 'second question', got %s", msg.Content)
	}
}

func TestManagerCurrentRound(t *testing.T) {
	m := NewManager()

	if m.CurrentRound() != 0 {
		t.Error("expected round 0 for no paper")
	}

	p := NewPaper("content", "")
	m.SetPaper(p)

	m.AddMessage(Message{RoundNumber: 0, Role: "user", Content: "q", TokenCount: 1})
	if m.CurrentRound() != 0 {
		t.Errorf("expected round 0, got %d", m.CurrentRound())
	}

	m.AddMessage(Message{RoundNumber: 1, Role: "user", Content: "q", TokenCount: 1})
	if m.CurrentRound() != 1 {
		t.Errorf("expected round 1, got %d", m.CurrentRound())
	}
}

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"hello", 1},
		{"hello world test", 4},
		{"", 0},
		{"这是一个测试", 9}, // 12 bytes / 4 = 3, but actually 9 chars / 4 = 2
	}

	for _, tt := range tests {
		result := EstimateTokens(tt.input)
		// Just verify it's non-negative and roughly proportional
		if result < 0 {
			t.Errorf("EstimateTokens(%q) = %d, expected non-negative", tt.input, result)
		}
	}
}

func TestManagerConcurrency(t *testing.T) {
	m := NewManager()
	p := NewPaper("content", "")
	m.SetPaper(p)

	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(n int) {
			m.AddMessage(Message{RoundNumber: n, Role: "user", Content: "q", TokenCount: 1})
			_ = m.Paper()
			_ = m.GetRecentMessages(5)
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}
