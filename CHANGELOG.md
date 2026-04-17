# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.4.0] - 2026-04-17

### Added
- Pod **Status** column that mirrors kubectl logic — surfaces OOMKilled, CrashLoopBackOff, Init errors, Terminating, and non-zero exit codes from container statuses
- Pod **Restarts** column showing total restart count across all containers
- Real-time age refresh (1-second tick) that recomputes age strings client-side from cached timestamps — no API calls, safe at 1M+ pods
- Day-level age formatting (`7d`) for resources older than 24 hours
- `DefaultAgeRefreshInterval` config constant for the refresh tick interval

### Changed
- Pod column order: Name → Status → Restarts → Age → Namespace → Pod IP → Node
- Renamed pod "Phase" column to "Status" with proper container-aware status resolution
- Extracted pod status logic into dedicated `internal/k8s/pod_status.go` with clean helper functions
- Watch goroutine maintains `creationTimes` cache in sync with resources for zero-allocation age updates
- Coordinated sorting via `sort.Interface` (`resourceSorter`) instead of temporary index slice allocations
- Resolver template parsing errors are now handled explicitly instead of panicking

### Fixed
- Pod phase showing "Running" when containers are OOMKilled or in CrashLoopBackOff
- Missing restart count visibility despite pods restarting
- Age column showing stale values (e.g. "4s" forever) because it was only computed once at resolve time
- Age showing `<unknown>` instead of panicking on zero-value timestamps

## [0.3.1] - 2026-03-27

### Fixed
- Fix version number not being set correctly in release builds

### Changed
- Bump Go dependency group (6 updates)

## [0.3.0] - 2026-03-27

### Added
- Log filtering/search in streaming log view with match highlighting
- `FilterLogs` key binding for interactive log filtering
- Color-coded describe output — status values, timestamps, and keys are syntax-highlighted
- Color-coded pod phase in table view based on pod state
- Unit tests for log filtering functionality

### Fixed
- Reset filter state when switching between pods in log view
- Linter errors across the codebase

### Changed
- CI workflow split into separate linting and testing jobs
- Enhanced test command with timeout and count options

## [0.2.0] - 2026-02-24

### Added
- Edit Kubernetes resources with `e` key binding (opens in `$EDITOR`)
- View resource YAML with `y` key binding
- Switch cluster namespaces interactively
- Switch cluster contexts with `:ctx` command
- Help modal with `?` key binding
- Describe resources with scrollable viewport
- Live resource updates in the table view (resource watcher)
- Navigation history for drill-down between resources
- Container log viewing with `Enter`
- Save/copy container logs with `:cplogs` command
- Filter pods and services by namespace
- Services view with dynamic columns
- Cluster info display in header
- Reconnect to cluster capability
- Dynamic schema support for all resource types (view any Kubernetes resource)
- Dynamic client using the Kubernetes discovery API
- Structured logging with configurable log level (`--log-level` flag)
- Easter egg: Kitten Climber platformer game
- Discord community link in footer
- Open default namespace from `~/.k10s.conf` config
- Homebrew installation via `brew tap shvbsle/tap && brew install k10s`
- RPM and DEB packages for Linux distributions

### Changed
- Upgraded Bubble Tea from v1 to v2
- Improved `:resource` command with better UX
- Improved table sizing with `auto` page size
- Improved pagination style (Bubbles style with n/M format beyond 5 pages)
- `g`/`G` bindings in logs navigate to head/tail
- `Shift+J`/`Shift+K` (or `Shift+↑`/`Shift+↓`) jump to top/bottom of current page
- Continuous pagination: automatically advances page when cursor reaches boundary
- `q` key no longer exits the TUI (use `:q` or `Ctrl+C`)
- Describe view converted to scrollable viewport

### Fixed
- Handle panic when switching between clusters
- Null Object Pattern for disconnected client state (no more nil pointer panics)
- Refresh resources when reconnecting to a cluster
- Prevent double-processing of navigation keys
- Fixed indentation bug when log lines wrap
- Open namespace defined in config correctly

## [0.1.1] - 2026-01-31

### Fixed
- Configure Homebrew tap token and remove Scoop support

## [0.1.0] - 2026-01-31

### Added
- Initial release of k10s
- Paginated table view for Kubernetes resources
- Support for viewing pods, nodes, and namespaces
- Vim-like keybindings (j/k for navigation, h/l for pages, g/G for jump)
- Command mode with `:` key
- Configurable page size via `~/.k10s.conf`
- Customizable ASCII logo
- Built with Bubble Tea TUI framework

[Unreleased]: https://github.com/shvbsle/k10s/compare/v0.4.0...HEAD
[0.4.0]: https://github.com/shvbsle/k10s/compare/v0.3.1...v0.4.0
[0.3.1]: https://github.com/shvbsle/k10s/compare/v0.3.0...v0.3.1
[0.3.0]: https://github.com/shvbsle/k10s/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/shvbsle/k10s/compare/v0.1.1...v0.2.0
[0.1.1]: https://github.com/shvbsle/k10s/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/shvbsle/k10s/releases/tag/v0.1.0
