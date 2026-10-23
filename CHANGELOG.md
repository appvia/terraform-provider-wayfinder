# Changelog

All notable changes to this provider are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and the project uses
[Semantic Versioning](https://semver.org/).

Each release records the Wayfinder release it was built from.

## Unreleased

## 0.4.0 - 2026-10-23

Built from Wayfinder release `v3.4.0`.

### Added

- `use_cli` and `cli_profile` provider attributes: authenticate with the signed-in `wf` CLI session, with automatic token refresh.
- `publish_state` on `wayfinder_cloud_resource_plan` and `wayfinder_kube_resource_plan` (and their data sources): set `Published`, `Draft` or `Deprecated` to control whether a plan version is offered in the catalogue. Leave it unset to keep the state Wayfinder holds, for example one set with `wf publish`.

### Changed

- With no credentials configured, the provider now uses the `wf` CLI session (`use_cli` defaults to `true`); set `use_cli = false` to keep the previous behaviour.
- `token` / `WAYFINDER_TOKEN` accepts `wf` refresh tokens and refreshes them.

### Fixed

- Updates no longer round integers above 2^53 that Terraform does not manage (for example in feature integration parameters) when writing an object back to Wayfinder.
