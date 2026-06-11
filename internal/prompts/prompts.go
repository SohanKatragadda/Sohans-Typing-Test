package prompts

import (
	"math/rand"
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

var pool = map[Language][]string{
	English: {
		"calm focus turns a small practice session into a reliable habit that follows you through the rest of the day.",
		"typing rewards patience because each clean word gives your hands a better map of the next one.",
		"minimal tools are often the sharpest when they remove every distraction from the work in front of you.",
	},
	Python: {
		"def normalize_scores(scores):\n    total = sum(scores)\n    return [score / total for score in scores if total > 0]",
		"from pathlib import Path\n\nfor path in Path(\".\").glob(\"*.txt\"):\n    print(path.name, path.stat().st_size)",
		"class Timer:\n    def __init__(self, seconds):\n        self.seconds = seconds\n        self.remaining = seconds",
	},
	Java: {
		"public class Counter {\n    private int value;\n    public void increment() {\n        value++;\n    }\n}",
		"List<String> names = users.stream()\n    .map(User::name)\n    .filter(name -> !name.isBlank())\n    .toList();",
		"try {\n    Files.writeString(path, content);\n} catch (IOException error) {\n    logger.error(error.getMessage());\n}",
	},
	C: {
		"#include <stdio.h>\n\nint main(void) {\n    printf(\"ready\\n\");\n    return 0;\n}",
		"for (size_t i = 0; i < count; i++) {\n    total += values[i];\n}\nprintf(\"%zu\\n\", total);",
		"char buffer[128];\nif (fgets(buffer, sizeof buffer, stdin) != NULL) {\n    puts(buffer);\n}",
	},
	JavaScript: {
		"const totals = orders\n  .filter(order => order.paid)\n  .map(order => order.amount)\n  .reduce((sum, amount) => sum + amount, 0);",
		"async function loadUser(id) {\n  const response = await fetch(`/api/users/${id}`);\n  return response.json();\n}",
		"const button = document.querySelector(\"button\");\nbutton.addEventListener(\"click\", () => {\n  console.log(\"saved\");\n});",
	},
}

func Random(language Language, minRunes int) string {
	options := pool[language]
	if len(options) == 0 {
		options = pool[English]
	}

	var builder strings.Builder
	for builder.Len() < minRunes {
		if builder.Len() > 0 {
			builder.WriteString("\n\n")
		}
		builder.WriteString(options[rand.Intn(len(options))])
	}
	return builder.String()
}
