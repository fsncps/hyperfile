# Drag-and-Drop (External) Design

**Date:** 2026-03-03

## Goal

Allow the user to drag files from hyperfile into external X11/Wayland applications (email, Telegram, browsers, etc.) using a keybind that spawns a DnD helper tool.

## Hotkey Changes

- `delete_items` default rebinds from `['ctrl+d']` to `['delete']`
- New `drag_items = ['ctrl+d']` added to hotkeys.toml

## Config Option

In `config.toml`:

```toml
dnd_tool = "dragon"
```

- X11 users: `"dragon"` (https://github.com/mwh/dragon)
- Wayland users: `"ripdrag"` (https://github.com/nik012003/ripdrag)
- Power users: any compatible tool name

## Behaviour

**What gets dragged:** if the tree panel has a selection (`HasSelection()`), all selected paths are passed; otherwise the cursor node's path is used.

**Execution:** `cmd.Start()` — the helper runs alongside the TUI without suspending it.

**Flag logic (in code):**
- `"dragon"` → `dragon --and-exit <paths...>`
- `"ripdrag"` → `ripdrag <paths...>`
- anything else → `<tool> <paths...>`

**Error handling:** if the binary is not found (`exec.LookPath` fails), show a notify modal: "dnd_tool not found: <tool>".

## Files to Touch

| File | Change |
|---|---|
| `src/hyperfile_config/hotkeys.toml` | Rebind delete, add drag_items |
| `src/hyperfile_config/config.toml` | Add `dnd_tool = "dragon"` |
| `src/internal/common/config_type.go` | Add `DNDTool string` to `AppConfig`; `DragItems []string` to `HotkeysConfig` |
| `src/internal/handle_file_operations.go` | New `dragItems(tree)` function |
| `src/internal/handle_tree_panel.go` | Wire `DragItems` hotkey case |
