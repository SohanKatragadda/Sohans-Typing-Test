package prompts

import (
	"strings"
	"testing"
)

func TestProgrammingPromptPoolsUseFourSpaceIndentation(t *testing.T) {
	for _, language := range []Language{Python, Java, C, JavaScript} {
		for i, segment := range pool[language] {
			if strings.Contains(segment, "\t") {
				t.Fatalf("%s segment %d includes a tab: %q", language, i, segment)
			}
			if !strings.Contains(segment, "    ") {
				t.Fatalf("%s segment %d does not include four-space indentation: %q", language, i, segment)
			}
		}
	}
}

func TestPromptPoolsLoadFromLanguageFiles(t *testing.T) {
	for _, language := range Languages {
		if len(pool[language]) < 8 {
			t.Fatalf("%s prompt count = %d, want at least 8", language, len(pool[language]))
		}
	}
}

func TestPromptPoolsDoNotContainDuplicates(t *testing.T) {
	for _, language := range Languages {
		seen := map[string]bool{}
		for _, prompt := range pool[language] {
			if seen[prompt] {
				t.Fatalf("%s contains duplicate prompt: %q", language, prompt)
			}
			seen[prompt] = true
		}
	}
}

func TestPromptLineDecoding(t *testing.T) {
	got := decodePromptLine(`def f():\n    return "ok"`)
	want := "def f():\n    return \"ok\""

	if got != want {
		t.Fatalf("decoded prompt = %q, want %q", got, want)
	}

	c := decodePromptLine(`printf("ready\\n");`)
	if c != `printf("ready\n");` {
		t.Fatalf("literal slash escape should survive, got %q", c)
	}
}

func TestMoreExcludingAvoidsExistingPromptsWhenAvailable(t *testing.T) {
	excluded := pool[English][:3]
	got := MoreExcluding(English, 3, excluded)

	for _, prompt := range got {
		for _, blocked := range excluded {
			if prompt == blocked {
				t.Fatalf("prompt %q should have been excluded", prompt)
			}
		}
	}

	seen := map[string]bool{}
	for _, prompt := range got {
		if seen[prompt] {
			t.Fatalf("duplicate prompt returned: %q", prompt)
		}
		seen[prompt] = true
	}
}
