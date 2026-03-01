# Hyperfile Keybindings Reference

Config file: `~/.config/hyperfile/hotkeys.toml`

**Design rule:** bare letters, digits, and filename characters (`[A-Za-z0-9@_.+=/-]`)
are reserved for the always-on filter input. Every configurable hotkey carries a modifier.

---

## Hard-coded (not in config, source changes required)

| Key | Action |
|-----|--------|
| `enter` | Confirm / open item |
| `esc` | Cancel / close modal |
| `?` | Open help |
| `:` | Open shell command prompt |
| `>` | Open hyperfile command prompt |
| `shift+up` | Extend selection up |
| `shift+down` | Extend selection down |
| Mouse wheel | Scroll in focused panel |

---

## Configurable via hotkeys.toml

### Navigation — arrows and special keys, no modifier needed

| Default | Config key | Action |
|---|---|---|
| `up` | `list_up` | Move cursor up |
| `down` | `list_down` | Move cursor down |
| `pgup` | `page_up` | Page up |
| `pgdown` | `page_down` | Page down |
| `left`, `backspace` | `parent_directory` | Go to parent directory |
| `right` | `confirm` | Open item (enter is also hard-coded) |
| `tab` | `next_file_panel` | Focus next file panel |
| `shift+tab` | `previous_file_panel` | Focus previous file panel |

### File operations — `ctrl+` acts on files

| Default | Config key | Action |
|---|---|---|
| `ctrl+n` | `file_panel_item_create` | Create new file or directory |
| `ctrl+r` | `file_panel_item_rename` | Rename item |
| `ctrl+c` | `copy_items` | Copy selected items |
| `ctrl+x` | `cut_items` | Cut selected items |
| `ctrl+v` | `paste_items` | Paste items |
| `ctrl+d` | `delete_items` | Delete items (to trash) |
| `ctrl+e` | `extract_file` | Extract archive |
| `ctrl+a` | `compress_file` | Compress to archive |
| `ctrl+p` | `copy_path` | Copy selected item's path to clipboard |

### App / UI operations — `alt+` navigates the app itself

| Default | Config key | Action |
|---|---|---|
| `ctrl+q` | `quit` | Quit |
| `alt+n` | `create_new_file_panel` | Open new file panel |
| `alt+w` | `close_file_panel` | Close current file panel |
| `alt+p` | `focus_on_process_bar` | Focus process bar |
| `alt+s` | `focus_on_sidebar` | Focus sidebar |
| `alt+m` | `focus_on_metadata` | Focus metadata panel |
| `alt+f` | `toggle_file_preview_panel` | Toggle file preview panel |
| `alt+o` | `open_sort_options_menu` | Open sort options menu |
| `alt+r` | `toggle_reverse_sort` | Toggle reverse sort |
| `alt+.` | `toggle_dot_file` | Toggle hidden files |
| `alt+e` | `open_file_with_editor` | Open file in $EDITOR |
| `alt+E` | `open_current_directory_with_editor` | Open current dir in $EDITOR |
| `alt+b` | `pinned_directory` | Pin / unpin directory (bookmark) |
| `alt+t` | `toggle_footer` | Toggle footer bar |
| `alt+c` | `copy_present_working_directory` | Copy current directory path to clipboard |

### Selection — no dedicated mode, works anywhere

| Default | Config key | Action |
|---|---|---|
| `shift+up` *(hard-coded)* | `file_panel_select_mode_items_select_up` | Extend selection up |
| `shift+down` *(hard-coded)* | `file_panel_select_mode_items_select_down` | Extend selection down |
| `alt+a` | `file_panel_select_all_items` | Select all items |

---

## Pending source changes

The following are still read from `hotkeys.toml` but will be hard-coded in source
once the refactor is complete, at which point they'll be removed from the config file:

`confirm_typing`, `cancel_typing`, `open_help_menu`, `open_command_line`, `open_spf_prompt`, `search_bar`

Visual mode (`change_panel_mode`) has been removed — `change_panel_mode = []` in config.
