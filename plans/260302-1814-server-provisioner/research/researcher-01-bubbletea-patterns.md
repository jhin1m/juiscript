# Research: Bubble Tea TUI Patterns for Interactive Checklist + Install Screens

Date: 2026-03-02
Sources: charmbracelet/bubbletea docs, pkg.go.dev, charm.land blog, leg100 tips, inngest blog

---

## 1. Multi-Select Checklist Pattern

Model holds cursor index and a `map[int]bool` for selected items. Space toggles, Enter confirms.

```go
type model struct {
    items    []string
    cursor   int
    selected map[int]bool
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "up", "k":
            if m.cursor > 0 { m.cursor-- }
        case "down", "j":
            if m.cursor < len(m.items)-1 { m.cursor++ }
        case " ":
            m.selected[m.cursor] = !m.selected[m.cursor]
        case "enter":
            return m, startInstall(m.selectedItems())
        }
    }
    return m, nil
}

func (m model) View() string {
    var b strings.Builder
    for i, item := range m.items {
        cursor := " "
        if m.cursor == i { cursor = ">" }
        check := "[ ]"
        if m.selected[i] { check = "[x]" }
        fmt.Fprintf(&b, "%s %s %s\n", cursor, check, item)
    }
    return b.String()
}
```

**Alternative**: Use `github.com/charmbracelet/bubbles/list` with a custom delegate for richer UX (filtering, pagination). For simple checklists, the manual map approach is KISS-compliant.

---

## 2. Spinner Pattern During Long-Running Operations

Use `github.com/charmbracelet/bubbles/spinner`.

```go
import "github.com/charmbracelet/bubbles/spinner"

type model struct {
    spinner spinner.Model
    state   appState
    output  []string
}

func initialModel() model {
    s := spinner.New()
    s.Spinner = spinner.Dot  // or MiniDot, Line, Jump, Pulse, Points, Globe, Moon, Monkey
    return model{spinner: s}
}

func (m model) Init() tea.Cmd {
    return m.spinner.Tick  // required to animate
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case spinner.TickMsg:
        var cmd tea.Cmd
        m.spinner, cmd = m.spinner.Update(msg)
        return m, cmd
    }
    return m, nil
}

func (m model) View() string {
    if m.state == stateInstalling {
        return fmt.Sprintf("%s Installing...\n", m.spinner.View())
    }
    return ""
}
```

Key: `Init()` must return `m.spinner.Tick` or spinner won't animate.

---

## 3. State Machine Pattern (checklist → installing → done)

Use an `int` or custom type as state enum. Route `Update` and `View` based on it.

```go
type appState int
const (
    stateSelect appState = iota
    stateInstalling
    stateDone
    stateError
)

type model struct {
    state   appState
    items   []string
    selected map[int]bool
    cursor  int
    spinner spinner.Model
    output  []string
    err     error
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch m.state {
    case stateSelect:
        return m.updateSelect(msg)
    case stateInstalling:
        return m.updateInstalling(msg)
    case stateDone, stateError:
        // only handle quit
        if k, ok := msg.(tea.KeyMsg); ok && k.String() == "q" {
            return m, tea.Quit
        }
    }
    return m, nil
}

func (m model) View() string {
    switch m.state {
    case stateSelect:    return m.viewSelect()
    case stateInstalling: return m.viewInstalling()
    case stateDone:      return m.viewDone()
    case stateError:     return fmt.Sprintf("Error: %v\nPress q to quit.\n", m.err)
    }
    return ""
}
```

---

## 4. Running Background Commands + Streaming Output

**The golden rule**: Never use raw goroutines. Wrap in `tea.Cmd` (which runs in a managed goroutine).

### Pattern: Read stdout line-by-line via channel + tea.Cmd polling

```go
// Message types
type cmdOutputMsg string
type cmdDoneMsg  struct{ err error }

// Start command, stream stdout via channel
func runInstall(pkgs []string) tea.Cmd {
    return func() tea.Msg {
        cmd := exec.Command("apt-get", append([]string{"install", "-y"}, pkgs...)...)
        stdout, _ := cmd.StdoutPipe()
        cmd.Stderr = cmd.Stdout  // merge stderr into stdout pipe is NOT possible this way
        // Better: use combined output reader
        cmd.Start()
        scanner := bufio.NewScanner(stdout)
        // This blocks — only returns ONE message (last line), not streaming
        // For true streaming, use channel approach below
        var last string
        for scanner.Scan() { last = scanner.Text() }
        err := cmd.Wait()
        return cmdDoneMsg{err}
    }
}
```

**True streaming pattern** (send each line as a message):

```go
var outputCh = make(chan string, 100)

func startCmd(pkgs []string) tea.Cmd {
    return func() tea.Msg {
        cmd := exec.Command("apt-get", append([]string{"install", "-y"}, pkgs...)...)
        pr, pw, _ := os.Pipe()
        cmd.Stdout = pw
        cmd.Stderr = pw
        cmd.Start()
        go func() {
            scanner := bufio.NewScanner(pr)
            for scanner.Scan() {
                outputCh <- scanner.Text()
            }
            pw.Close()
            cmd.Wait()
            close(outputCh)
        }()
        return cmdStartedMsg{}
    }
}

// Poll channel via tea.Cmd to bring lines into Update()
func waitForOutput() tea.Cmd {
    return func() tea.Msg {
        line, ok := <-outputCh
        if !ok { return cmdDoneMsg{} }
        return cmdOutputMsg(line)
    }
}

// In Update, after receiving cmdOutputMsg:
case cmdOutputMsg:
    m.output = append(m.output, string(msg))
    return m, waitForOutput()  // re-schedule to read next line
```

Key insight: each `tea.Cmd` returns exactly ONE `tea.Msg`. To stream N lines, chain: each receipt of `cmdOutputMsg` schedules another `waitForOutput()`.

---

## 5. Best Practices for tea.Cmd with Long-Running Processes

| Rule | Rationale |
|------|-----------|
| Never block `Update()` or `View()` | They run on the main loop; blocking freezes render |
| Never use raw `go func()` | Use `tea.Cmd` wrapper — program manages goroutine lifecycle |
| `tea.Batch()` for concurrent cmds | Runs multiple Cmds in parallel, results arrive unordered |
| `tea.Sequence()` for ordered cmds | Runs Cmds one-at-a-time in order |
| Chain Cmds for streaming | Each `tea.Cmd` returns one msg; re-issue from `Update` to continue |
| `tea.ExecProcess()` for interactive | Pauses TUI, hands terminal to subprocess (e.g., vim, ssh), resumes after |
| Keep `View()` pure/fast | No I/O, no side effects — just string rendering from model state |
| Use `tea.Tick()` for progress polling | When you can't stream, tick every N ms and check status |

### ExecProcess (blocking, interactive subprocess)
```go
// Hands terminal control to subprocess — TUI pauses
cmd := tea.ExecProcess(exec.Command("vim", "file.txt"), func(err error) tea.Msg {
    return editorDoneMsg{err}
})
```
Not suitable for streaming — use channel pattern above for apt-get.

---

## Key Libraries

- `github.com/charmbracelet/bubbletea` — core framework (v0.25+, v2 in dev)
- `github.com/charmbracelet/bubbles` — spinner, list, textinput, viewport, progress
- `github.com/charmbracelet/lipgloss` — styling, layout math

---

## Recommended Component Stack

```
stateSelect:     manual []item + map[int]bool  (KISS, no bubbles/list needed)
stateInstalling: bubbles/spinner + channel stream → viewport for scrolling output
stateDone:       plain View() string
```

---

## Unresolved Questions

1. Does bubbletea v2 (if released) change the `tea.Cmd` / streaming API significantly?
2. `os.Pipe()` + goroutine inside `tea.Cmd` technically violates "no raw goroutines" — is `program.Send()` from goroutine a cleaner alternative? (It is: `p.Send(msg)` thread-safely injects msgs from outside the program loop.)
3. For `apt-get` specifically: does `-y` + stdout merge require `CombinedOutput()` or pipe trick? Need to verify behavior under sudo/PTY.
