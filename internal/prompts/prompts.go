package prompts

import (
	"embed"
	"math/rand"
	"strconv"
	"strings"
)

type Language string

const (
	English    Language = "english"
	Python     Language = "python"
	Java       Language = "java"
	C          Language = "c"
	JavaScript Language = "javascript"
)

var Languages = []Language{English, Python, Java, C, JavaScript}

//go:embed data/*.txt
var promptFiles embed.FS

var promptFileByLanguage = map[Language]string{
	English:    "data/english.txt",
	Python:     "data/python.txt",
	Java:       "data/java.txt",
	C:          "data/c.txt",
	JavaScript: "data/javascript.txt",
}

var pool = loadPools()

func Generate(language Language, durationSeconds int) []string {
	return More(language, initialCount(durationSeconds))
}

func More(language Language, count int) []string {
	return MoreExcluding(language, count, nil)
}

func MoreExcluding(language Language, count int, exclude []string) []string {
	options := pool[language]
	if len(options) == 0 {
		options = pool[English]
	}
	if count < 1 {
		count = 1
	}

	excluded := make(map[string]bool, len(exclude))
	for _, prompt := range exclude {
		excluded[prompt] = true
	}

	available := make([]string, 0, len(options))
	for _, option := range options {
		if !excluded[option] {
			available = append(available, option)
		}
	}
	if len(available) == 0 {
		available = append(available, options...)
	}

	rand.Shuffle(len(available), func(i, j int) {
		available[i], available[j] = available[j], available[i]
	})

	segments := make([]string, 0, count)
	for len(segments) < count {
		for _, option := range available {
			segments = append(segments, option)
			if len(segments) == count {
				break
			}
		}
		if len(segments) < count {
			available = append([]string(nil), options...)
			rand.Shuffle(len(available), func(i, j int) {
				available[i], available[j] = available[j], available[i]
			})
		}
	}
	return segments
}

func loadPools() map[Language][]string {
	pools := make(map[Language][]string, len(promptFileByLanguage))
	for language, path := range promptFileByLanguage {
		content, err := promptFiles.ReadFile(path)
		if err != nil {
			continue
		}
		pools[language] = parsePromptLines(string(content))
	}
	return pools
}

func parsePromptLines(content string) []string {
	lines := strings.Split(content, "\n")
	prompts := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		prompts = append(prompts, decodePromptLine(line))
	}
	return prompts
}

func decodePromptLine(line string) string {
	decoded, err := strconv.Unquote(`"` + strings.ReplaceAll(line, `"`, `\"`) + `"`)
	if err != nil {
		return line
	}
	return decoded
}

func initialCount(durationSeconds int) int {
	switch {
	case durationSeconds <= 15:
		return 8
	case durationSeconds <= 30:
		return 14
	case durationSeconds <= 60:
		return 24
	default:
		return 44
	}
}
