# Things that need to be done

## Core Features

- [ ] **Search for pods/resources** ([#60](https://github.com/shvbsle/k10s/issues/60)) - Triggered by `/` keybinding. Opens a search box with live filtering - as the user types, the current view updates in real-time to show matching results
- [x] **Switch between cluster contexts** ([#61](https://github.com/shvbsle/k10s/issues/61)) - Triggered by `:ctx` in command mode. Opens a context selector to switch between different kubeconfig contexts without leaving the TUI
- [x] **Edit manifests** ([#62](https://github.com/shvbsle/k10s/issues/62)) - Triggered by `e` keybinding when cursor is on any resource. Opens the resource manifest in the user's default editor (e.g., vim). Saving and closing the editor automatically applies the changes to the cluster
- [x] **Describe output for resources** ([#63](https://github.com/shvbsle/k10s/issues/63)) - Triggered by `d` keybinding when cursor is on any resource. Renders `kubectl describe` output for the selected resource
- [x] **YAML view for resources** - Triggered by `y` keybinding when cursor is on any resource. Renders the resource YAML manifest with syntax highlighting (bold keys), scrollable with arrow keys, line number toggle (`n`), and line wrapping toggle (`w`)
- [ ] **Syntax highlighting for describe output** ([#64](https://github.com/shvbsle/k10s/issues/64)) - Add syntax highlighting to the describe output view for better readability
- [ ] **SSH into containers** ([#65](https://github.com/shvbsle/k10s/issues/65)) - Triggered by `s` keybinding when cursor is on any resource. Equivalent to `kubectl exec -it <pod> -- /bin/sh`
- [ ] **Logs shortcut** ([#66](https://github.com/shvbsle/k10s/issues/66)) - Triggered by `l` keybinding to display logs for selected pod/container
- [ ] **Real-time logs** ([#67](https://github.com/shvbsle/k10s/issues/67)) - Stream logs in real-time with auto-scroll and filtering
- [x] **Switch namespaces** ([]()) - Use `:ns` command to switch between namespaces


## Developer Experience

- [ ] **Faster builds + tests** ([#68](https://github.com/shvbsle/k10s/issues/68)) - Optimize build pipeline and test execution time

## User Experience

- [x] **Help page** ([#69](https://github.com/shvbsle/k10s/issues/69)) - Triggered by `?` keybinding. Shows all available commands and keybindings
- [ ] **Shortcuts/profiles for keybindings** ([#70](https://github.com/shvbsle/k10s/issues/70)) - Customizable keybinding profiles (similar to nvim configuration)
- [ ] **Scrollable logs view** ([#73](https://github.com/shvbsle/k10s/issues/73)) - Replace pagination with vertical scrolling for logs view
- [x] **Scrollable describe view** ([#74](https://github.com/shvbsle/k10s/issues/74)) - Replace pagination with vertical scrolling for describe output
- [ ] **Cloud cluster context switching** () - Improved context switching for cloud environments (EKS, GKE, AKS). On cloud desktops, clusters across multiple accounts may have different credential requirements. This feature would streamline authentication and context switching for multi-account cloud setups.
- [ ] **Table view aesthetic improvements** () - Enhance the table view for better readability and data visibility:
  1. Color-coded namespace column - assign consistent colors to different namespaces to visually distinguish them
  2. Color-coded phase/status column - green for Running, yellow for Pending, red for error states, gray for Completed
  3. Horizontal scroll for table - use `<`/`>` or arrow keys to scroll horizontally when columns are clipped (e.g., Node column), allowing users to see all data without truncation

## Branding

- [ ] **Design k10s logo** ([#71](https://github.com/shvbsle/k10s/issues/71)) - Create a logo for the project
