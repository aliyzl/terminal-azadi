---
status: diagnosed
trigger: "Paste (Cmd+V / Ctrl+V) doesn't work in TUI text input modals for Add Server and Add Subscription"
created: 2026-02-25T00:00:00Z
updated: 2026-02-25T00:00:00Z
---

## Current Focus

hypothesis: CONFIRMED - Two independent root causes block paste from reaching the textinput
test: Code trace complete
expecting: N/A
next_action: Deliver diagnosis

## Symptoms

expected: When add-server or add-subscription modal is open, user can paste a URI/URL into the text input field via Cmd+V or Ctrl+V.
actual: Modal opens but paste does not insert text into the text input.
errors: None reported (silent failure).
reproduction: Open TUI, press 'a' or 's' to open input modal, attempt Ctrl+V or Cmd+V.
started: Since implementation (never worked).

## Eliminated

- hypothesis: textinput not focused
  evidence: SetMode() calls m.textInput.Focus() (input.go:57) and returns the resulting cmd. The root model dispatches this cmd (app.go:293, 297). Focus is correctly set.
  timestamp: 2026-02-25

- hypothesis: keys.go defines a conflicting binding that intercepts ctrl+v / cmd+v
  evidence: keys.go has no binding for ctrl+v, cmd+v, or any paste-related key. No interception at the keyMap level.
  timestamp: 2026-02-25

## Evidence

- timestamp: 2026-02-25
  checked: Bubble Tea v2 paste message types (bubbletea/v2@v2.0.0/paste.go)
  found: BT v2 introduces `tea.PasteMsg`, `tea.PasteStartMsg`, `tea.PasteEndMsg` as dedicated message types separate from `tea.KeyPressMsg`. Bracketed paste is enabled by default. When the terminal receives a bracketed paste sequence, BT v2 emits a `tea.PasteMsg{Content: "..."}` -- NOT a `tea.KeyPressMsg`.
  implication: The paste content arrives as `tea.PasteMsg`, which is a completely different type from `tea.KeyPressMsg`.

- timestamp: 2026-02-25
  checked: textinput Update method (bubbles/v2@v2.0.0/textinput/textinput.go:580-662)
  found: textinput.Update handles THREE paste paths -- (1) `tea.KeyPressMsg` matching `m.KeyMap.Paste` (ctrl+v) which returns a `Paste` command that reads from system clipboard via `atotto/clipboard`, yielding an internal `pasteMsg`; (2) `tea.PasteMsg` from bracketed paste, inserting `msg.Content` directly; (3) internal `pasteMsg` from the clipboard read command, inserting the string.
  implication: The textinput is fully capable of handling paste IF it receives the messages.

- timestamp: 2026-02-25
  checked: Root model Update switch in app.go:108-213
  found: The root model's type switch handles these cases in order: WindowSizeMsg, tickMsg, pingResultMsg, allPingsCompleteMsg, serverAddedMsg, serverRemovedMsg, subscriptionFetchedMsg, serversReplacedMsg, connectResultMsg, disconnectMsg, errMsg, tea.KeyPressMsg. The fallthrough (lines 205-212) passes unmatched messages to serverList only. There is NO case for `tea.PasteMsg`, `tea.PasteStartMsg`, `tea.PasteEndMsg`, `pasteMsg`, or `pasteErrMsg`.
  implication: `tea.PasteMsg` falls through to the default handler which sends it to `m.serverList.Update(msg)` instead of `m.input.Update(msg)`. The textinput never receives the paste content.

- timestamp: 2026-02-25
  checked: handleKeyPress routing for viewAddServer/viewAddSubscription (app.go:228-260)
  found: When view is viewAddServer or viewAddSubscription, the method matches on `msg.Keystroke()`. It handles "esc" and "enter" explicitly, then routes all other keys to `m.input.Update(msg)` via the `default` case. This works for `tea.KeyPressMsg` but `handleKeyPress` is ONLY called for `tea.KeyPressMsg` (line 201-202).
  implication: Even if ctrl+v arrives as a KeyPressMsg, the default case would correctly forward it to input.Update. But the COMMAND returned by textinput (the `Paste` function that reads from clipboard) would produce a `pasteMsg` (internal type) or `pasteErrMsg` -- and those are also not routed to the input model.

- timestamp: 2026-02-25
  checked: Ctrl+V flow specifically
  found: On macOS terminals, Cmd+V is intercepted by the terminal emulator and converted to a bracketed paste sequence (tea.PasteMsg). Ctrl+V in a terminal does NOT produce a paste -- it's a literal control character. However, textinput binds ctrl+v to its Paste key which uses atotto/clipboard (system clipboard read). So: (A) Cmd+V -> terminal paste -> tea.PasteMsg -> NOT routed to input model. (B) Ctrl+V -> tea.KeyPressMsg -> routed to input model -> textinput returns Paste command -> Paste command produces pasteMsg -> pasteMsg NOT routed to input model.
  implication: BOTH paste paths are broken. Path A fails at the root model's message routing. Path B fails at the command result routing.

## Resolution

root_cause: |
  Two bugs in app.go prevent paste from working in the input modals:

  **Bug 1 (Primary): `tea.PasteMsg` not routed to input model.**
  In Bubble Tea v2, when the user pastes via the terminal (Cmd+V on macOS, or any
  bracketed paste), the framework emits a `tea.PasteMsg` -- NOT a `tea.KeyPressMsg`.
  The root model's Update (app.go:108-213) has no case for `tea.PasteMsg`. It falls
  through to the default handler (lines 205-212) which only passes messages to
  `m.serverList`, never to `m.input`. The textinput never receives the pasted content.

  **Bug 2 (Secondary): Internal clipboard command result (`pasteMsg`) not routed.**
  Even if Ctrl+V were to reach textinput as a KeyPressMsg (which it does via the
  default case in handleKeyPress), textinput returns a `Paste` tea.Cmd that reads
  from the system clipboard via atotto/clipboard. This command produces an internal
  `pasteMsg` (or `pasteErrMsg`). These are unexported types from the textinput package,
  so the root model cannot match on them explicitly. However, they also fall through to
  the default handler and go to serverList instead of input. The textinput never
  receives the clipboard read result.

  Both bugs stem from the same architectural issue: the root model's fallthrough only
  forwards unhandled messages to `m.serverList`, ignoring `m.input` entirely for
  non-KeyPressMsg messages.

fix: |
  NOT APPLIED (diagnosis only). Recommended fix direction:

  **Option A (Targeted -- minimal change):**
  Add a `tea.PasteMsg` case to the root model's Update that forwards to `m.input.Update`
  when the view is viewAddServer or viewAddSubscription:

  ```go
  case tea.PasteMsg:
      if m.view == viewAddServer || m.view == viewAddSubscription {
          var cmd tea.Cmd
          m.input, cmd = m.input.Update(msg)
          return m, cmd
      }
      return m, nil
  ```

  And change the default fallthrough (lines 205-212) to also forward to input when
  in an input view state, so that pasteMsg/pasteErrMsg from the clipboard command
  also reach textinput:

  ```go
  // Pass other messages to the active child model
  var cmd tea.Cmd
  if m.view == viewAddServer || m.view == viewAddSubscription {
      m.input, cmd = m.input.Update(msg)
  } else {
      m.serverList, cmd = m.serverList.Update(msg)
  }
  ```

  **Option B (Broader -- better architecture):**
  Restructure the fallthrough to always route unhandled messages to whichever child
  model is active based on view state. This would fix paste AND prevent any future
  message routing bugs for new message types.

verification: Not applicable (diagnosis only).
files_changed: []
