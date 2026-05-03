package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/paperpaper/paperpaper/internal/config"
)

type Message struct {
	RoundNumber int       `json:"round_number"`
	Role        string    `json:"role"`
	Content     string    `json:"content"`
	Digest      string    `json:"digest,omitempty"`
	TokenCount  int       `json:"token_count"`
	CreatedAt   time.Time `json:"created_at"`
}

type Paper struct {
	ID             int       `json:"id"`
	Title          string    `json:"title"`
	SourceURL      string    `json:"source_url"`
	Content        string    `json:"content"`
	InitialSummary string    `json:"initial_summary"`
	ModelUsed      string    `json:"model_used"`
	TotalTokens    int       `json:"total_tokens_used"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	Messages       []Message `json:"messages"`
}

type Manager struct {
	mu    sync.Mutex
	paper *Paper
}

func NewManager() *Manager {
	return &Manager{}
}

func (m *Manager) Paper() *Paper {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.paper
}

func (m *Manager) SetPaper(p *Paper) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.paper = p
}

func (m *Manager) AddMessage(msg Message) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.paper != nil {
		m.paper.Messages = append(m.paper.Messages, msg)
		m.paper.UpdatedAt = time.Now()
		m.paper.TotalTokens += msg.TokenCount
	}
}

func (m *Manager) UpdateLastAssistant(content string, tokenCount int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.paper == nil || len(m.paper.Messages) == 0 {
		return
	}
	for i := len(m.paper.Messages) - 1; i >= 0; i-- {
		if m.paper.Messages[i].Role == "assistant" {
			m.paper.Messages[i].Content = content
			m.paper.Messages[i].TokenCount = tokenCount
			m.paper.UpdatedAt = time.Now()
			return
		}
	}
}

func (m *Manager) SetInitialSummary(summary string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.paper != nil {
		m.paper.InitialSummary = summary
		m.paper.UpdatedAt = time.Now()
	}
}

func (m *Manager) SetTitle(title string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.paper != nil {
		m.paper.Title = title
	}
}

func (m *Manager) GetRecentMessages(n int) []Message {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.paper == nil {
		return nil
	}
	msgs := m.paper.Messages
	if len(msgs) <= n*2 {
		return msgs
	}
	return msgs[len(msgs)-n*2:]
}

func (m *Manager) CurrentRound() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.paper == nil || len(m.paper.Messages) == 0 {
		return 0
	}
	return m.paper.Messages[len(m.paper.Messages)-1].RoundNumber
}

func (m *Manager) DeleteRound(round int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.paper == nil {
		return
	}
	var filtered []Message
	for _, msg := range m.paper.Messages {
		if msg.RoundNumber != round {
			filtered = append(filtered, msg)
		}
	}
	m.paper.Messages = filtered
	m.paper.UpdatedAt = time.Now()
}

func (m *Manager) DeleteLastRound() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.paper == nil || len(m.paper.Messages) == 0 {
		return
	}
	lastRound := m.paper.Messages[len(m.paper.Messages)-1].RoundNumber
	var filtered []Message
	for _, msg := range m.paper.Messages {
		if msg.RoundNumber != lastRound {
			filtered = append(filtered, msg)
		}
	}
	m.paper.Messages = filtered
	m.paper.UpdatedAt = time.Now()
}

func (m *Manager) GetLastUserMessage() *Message {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.paper == nil {
		return nil
	}
	for i := len(m.paper.Messages) - 1; i >= 0; i-- {
		if m.paper.Messages[i].Role == "user" {
			return &m.paper.Messages[i]
		}
	}
	return nil
}

func (m *Manager) Save() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.paper == nil {
		return nil
	}
	return SavePaper(m.paper)
}

// nextID returns the next available paper ID.
func nextID() int {
	dir := config.PapersDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 1
	}
	maxID := 0
	for _, e := range entries {
		var id int
		if _, err := fmt.Sscanf(e.Name(), "%d.json", &id); err == nil && id > maxID {
			maxID = id
		}
	}
	return maxID + 1
}

func NewPaper(content string, sourceURL string) *Paper {
	now := time.Now()
	return &Paper{
		ID:        nextID(),
		SourceURL: sourceURL,
		Content:   content,
		CreatedAt: now,
		UpdatedAt: now,
		Messages:  []Message{},
	}
}

func SavePaper(p *Paper) error {
	dir := config.PapersDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, fmt.Sprintf("%d.json", p.ID)), data, 0644)
}

func LoadPaper(id int) (*Paper, error) {
	path := filepath.Join(config.PapersDir(), fmt.Sprintf("%d.json", id))
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var p Paper
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

func DeletePaper(id int) error {
	path := filepath.Join(config.PapersDir(), fmt.Sprintf("%d.json", id))
	return os.Remove(path)
}

type PaperSummary struct {
	ID    int
	Title string
}

func ListPapers() ([]PaperSummary, error) {
	dir := config.PapersDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var papers []PaperSummary
	for _, e := range entries {
		var id int
		if _, err := fmt.Sscanf(e.Name(), "%d.json", &id); err != nil {
			continue
		}
		p, err := LoadPaper(id)
		if err != nil {
			continue
		}
		title := p.Title
		if title == "" {
			title = fmt.Sprintf("Paper #%d", p.ID)
		}
		papers = append(papers, PaperSummary{ID: p.ID, Title: title})
	}

	sort.Slice(papers, func(i, j int) bool {
		return papers[i].ID < papers[j].ID
	})

	return papers, nil
}

func EstimateTokens(text string) int {
	return len(text) / 4
}
