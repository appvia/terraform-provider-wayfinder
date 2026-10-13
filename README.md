# Terraform Provider for Wayfinder

Manage your Wayfinder SaaS configuration (workspaces, environments, variables, secrets, access and more) as code with Terraform or OpenTofu. This provider is under development. See [Contributing](#contributing) for build and contribution details.

Licensed under the Apache License 2.0.

## Compatibility

The provider is built from the Wayfinder release recorded in `schema/wayfinder-manifest.json` (`wayfinderVersion`).

## Contributing

Resources are generated from the Wayfinder release pinned in `WAYFINDER_REF`. Open issues rather than PRs that edit generated files. See [DEVELOPMENT.md](DEVELOPMENT.md) for how generation, overrides and the Wayfinder pin work.
