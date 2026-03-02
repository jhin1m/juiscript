# Bubble Tea TUI Patterns & Best Practices

**Date:** 2026-03-02 | **Focus:** Go CLI tools with Bubble Tea framework architecture

## 1. Core Architecture: The Elm Model

Bubble Tea implements **The Elm Architecture** with three fundamental methods:
- **Model**: Holds application state (single source of truth)
- **Update**: Handles events/messages, returns new model + commands
- **View**: Pure function that renders UI from current state

Message-driven flow: Event → Update → View → Render

## 2. Project Structure Patterns

### Single Model (Small Apps)
Keep flat structure with `main.go` → `model.go` → run

### Multi-Model (Large Apps) — **Recommended for scalable CLIs**
```
cmd/
  tui-app/main.go
internal/
  tui/
    models/
      root.go
      home.go
      detail.go
    views/
      home.go
      detail.go
    commands/
      fetch.go
      navigate.go
  storage/
    cache.go
```

**Pattern**: Root model becomes message router & screen compositor, delegates to child models (like Bubbles components). Each child model owns Init/Update/View.

## 3. Multi-Screen Navigation Patterns

### Stack-Based Navigation (bubbletea-nav)
- `nav.NewStack(homeScreen{})` initializes stack
- Screens implement `nav.Screen` interface (similar to tea.Model)
- Navigation: `nav.Push()` (add screen), `nav.Pop()` (remove screen)
- **Pros**: Natural browser-like back/forward, nested hierarchies
- **Use case**: Workflows with multiple steps (form → review → confirm)

### Router Pattern
- Root model maintains router state
- Messages route to active screen based on state
- Screens emit custom messages for navigation
- **Pros**: Explicit, easier to test
- **Use case**: Dashboard with multiple tabs/sections

### Focus Manager Pattern
Implements `nav.Focusable` (tea.Model + Focus/Blur methods):
- FocusManager delegates messages to focused field
- Blur/Focus handle UI state (highlight active component)
- **Use case**: Forms with multiple focusable inputs

## 4. Lip Gloss Styling Best Practices

### Theme Organization
Define base styles and inherit:
```go
var (
  baseStyle  = lipgloss.NewStyle().Padding(1)
  headerStyle = baseStyle.Bold(true).Foreground(color.Blue)
  panelStyle  = baseStyle.Border(lipgloss.RoundedBorder())
)
```

### Dynamic Theming
- Listen for `tea.BackgroundColorMsg` on init
- Use `lipgloss.LightDark(darkColor, lightColor)` for adaptive colors
- Prevents contrast issues across terminal backgrounds

### Layout Precision
- Use `lipgloss.Height()` / `lipgloss.Width()` for dynamic calculations
- Fill content lines to exact panel height for consistent borders
- Use same border style across all panels for visual cohesion
- Avoid hardcoded dimensions

### Performance
- Cache computed styles in model
- Avoid recalculating layout on every View() call
- Use `lipgloss.NewStyle()` once, store in variables

## 5. Charm Ecosystem Libraries

| Library | Purpose | Key Features |
|---------|---------|--------------|
| **Bubble Tea** | TUI framework | Elm architecture, event loop, message routing |
| **Bubbles** | UI components | TextInput, List, Spinner, Paginator, Textarea, Viewport |
| **Lip Gloss** | Styling/layout | Colors, borders, padding, alignment, theme support |
| **Huh** | Interactive forms | Form validation, field types, styled forms |
| **Glamour** | Markdown rendering | Terminal markdown with colors & formatting |
| **Harmonica** | Animations | Spring physics, smooth transitions |
| **BubbleZone** | Mouse events | Clickable zones, drag/drop support |

**Separation of concerns**: Bubble Tea = structure (HTML), Lip Gloss = style (CSS)

## 6. Popular Reference Implementations

### Bubble Tea Examples (92+ in repo)
Official tutorials cover: basics, forms, tables, viewports, commands, errors

### Community Tools Using Bubble Tea
- **Charm CLI ecosystem**: gum (shell scripting), charm (markdown)
- **Proposed lazygit redesign**: Issue #2705 discusses Charm.sh integration
- **Known tools**: Interactive Cobra CLIs, terminal dashboards, log viewers

## 7. Key Performance & Development Patterns

### Event Loop Best Practices
- **Keep Update() fast**: Offload expensive operations to `tea.Cmd` functions
- **Understand message ordering**: Concurrent commands arrive in unspecified order; use `tea.Sequence` for dependencies
- **Avoid blocking**: Don't block in Update/View; defer to Cmd goroutines

### Testing & Documentation
- **teatest**: Automate TUI testing by emulating input/output
- **VHS**: Record terminal sessions as GIFs/videos for docs
- **Debug logging**: Dump messages to file during development (use `spew`)

### Debugging Techniques
- Live reload: Use file watchers to rebuild/restart automatically
- Model state inspection: Log state transitions to file
- Message tracing: Record incoming messages for diagnosis

## 8. Component Composition (Bubbles)

Bubbles components are themselves tea.Model implementations:
- Embed Bubbles components in root model
- Root delegates Update/View to child components
- Children emit custom messages for parent coordination
- Example: List + TextInput for filtered search

**Key Bubbles Components**:
- **List**: Pagination, filtering, custom rendering
- **TextInput**: Unicode support, validation hooks, scrolling
- **Spinner**: Activity indication with custom frames
- **Paginator**: Offset/page calculations (used internally by List)
- **Textarea**: Multi-line input with scrolling
- **Viewport**: Scrollable content display

## 9. Command Execution Pattern

```go
// In Update():
return m, tea.Batch(
  fetchDataCmd(),      // async operation
  tickCmd(),          // timer
  logMessageCmd(),    // side effect
)

// Commands spawn in separate goroutines
// Results return as messages to Update()
```

Messages from commands arrive asynchronously; order is non-deterministic. Use `tea.Sequence` for ordered operations or refactor to handle out-of-order arrival.

---

## Unresolved Questions

1. How to handle deep undo/redo stacks with nav.Stack pattern?
2. Best practice for persistent state (cache/DB) integration with Bubble Tea?
3. Performance implications of large Lists (1000+ items) — pagination vs virtualization?
4. Recommended approach for inter-model communication beyond root routing?

---

## Sources

- [GitHub - Bubble Tea TUI Framework](https://github.com/charmbracelet/bubbletea)
- [Tips for Building Bubble Tea Programs](https://leg100.github.io/en/posts/building-bubbletea-programs/)
- [Bubble Tea Navigation Pattern](https://github.com/pgavlin/bubbletea-nav)
- [Charm Ecosystem Libraries](https://charm.land/libs/)
- [Bubbles Components Library](https://github.com/charmbracelet/bubbles)
- [Lip Gloss Styling Guide](https://github.com/charmbracelet/lipgloss)
- [Multi-Model Management in Bubble Tea](https://donderom.com/posts/managing-nested-models-with-bubble-tea/)
- [Building TUI with Bubble Tea](https://packagemain.tech/p/terminal-ui-bubble-tea)
- [Bubble Tea in DEV Community](https://dev.to/andyhaskell/intro-to-bubble-tea-in-go-21lg)
- [DepScore Bubble Tea Analysis](https://depscore.com/posts/2025-09-29-bubbletea/)
- [Inngest: Interactive CLIs with Bubbletea](https://www.inngest.com/blog/interactive-clis-with-bubbletea)
