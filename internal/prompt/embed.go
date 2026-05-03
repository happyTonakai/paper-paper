package prompt

import (
	_ "embed"
	"os"
	"path/filepath"

	"github.com/paperpaper/paperpaper/internal/config"
)

//go:embed prompts/heavy.txt
var HeavyPrompt string

//go:embed prompts/light.txt
var LightPrompt string

//go:embed prompts/digest.txt
var DigestPrompt string

// Get returns the prompt, checking user override first.
func Get(name string, fallback string) string {
	userPath := filepath.Join(config.PromptsDir(), name+".txt")
	data, err := os.ReadFile(userPath)
	if err == nil {
		return string(data)
	}
	return fallback
}

func GetHeavy() string  { return Get("heavy", HeavyPrompt) }
func GetLight() string  { return Get("light", LightPrompt) }
func GetDigest() string { return Get("digest", DigestPrompt) }
