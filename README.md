# Typing Test TUI

A minimal Monkeytype-inspired terminal typing test built with Go, Bubble Tea,
Bubbles, and Lip Gloss.

## Run

```sh
go run ./cmd/typing-test-tui
```

Completed tests are saved to `typing_history.csv` in the current working
directory.

## Controls

- Type to start a test.
- `backspace` edits while the test is active.
- `r` restarts the current test.
- `tab`, `left`, or `right` switches between Test and History when a test is not
  active.
- `,` and `.` change language before a run starts.
- `[` and `]` change duration before a run starts.
- `q` or `ctrl+c` quits.

## Modes

Languages: English, Python, Java, C, JavaScript.

Durations: 15, 30, 60, 120 seconds.
