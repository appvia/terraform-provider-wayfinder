# Releasing

This guide is for maintainers publishing the Wayfinder Terraform provider. A release is
a signed GitHub release produced by the [release workflow](.github/workflows/release.yml)
(GoReleaser, configured in [.goreleaser.yml](.goreleaser.yml)) when a `vX.Y.Z` tag is
pushed. The Terraform Registry and the OpenTofu registry then pick it up.

## One-off prerequisites

1. **Signing key.** Create an RSA GPG key (RSA is required by the registries; ECC keys are
   not accepted by the Terraform Registry). Keep the private key and passphrase in the
   team password manager.
2. **Repository secrets** (Settings > Secrets and variables > Actions):

   | Name | Used by | Purpose |
   |---|---|---|
   | `GPG_PRIVATE_KEY` | `release.yml` | ASCII-armored private key (`gpg --armor --export-secret-keys <fingerprint>`) |
   | `PASSPHRASE` | `release.yml` | Passphrase of that key |
   | `ORG_ACTIONS_APP_ID`, `ORG_ACTIONS_APP_SECRET` (organisation secrets) | `bump-wayfinder.yml` / `manifest.yml` | Client ID and private key of the appvia org GitHub App; each run mints a short-lived, contents-read token for `appvia/wayfinder`. The app must be installed on that repository |
   | `PROVIDER_PR_TOKEN` | `bump-wayfinder.yml` | Token allowed to open the automated Wayfinder bump pull request |
   | `ACC_WAYFINDER_SERVER`, `ACC_WAYFINDER_TENANT`, `ACC_WAYFINDER_CLIENT_ID`, `ACC_WAYFINDER_CLIENT_SECRET` | `acceptance.yml` | Service account credential for the test tenant |

   `GITHUB_TOKEN` is provided by GitHub. GoReleaser cannot use a passphrase-protected key
   directly, so the workflow imports the key into the runner's keyring with
   `crazy-max/ghaction-import-gpg` and passes the fingerprint on as `GPG_FINGERPRINT`.
3. **Terraform Registry.** Sign in to <https://registry.terraform.io> with the GitHub
   account that owns the `appvia` organisation. Under *User Settings > Signing Keys* add
   the **public** key (`gpg --armor --export <fingerprint>`) to the `appvia` namespace, then
   publish the provider (*Publish > Provider*, repository
   `appvia/terraform-provider-wayfinder`). Publishing adds a webhook to the GitHub
   repository; every release created afterwards is ingested automatically.
4. **OpenTofu registry.** The OpenTofu registry is driven by issue forms in
   <https://github.com/opentofu/registry> (checked against its README). Using the GitHub
   issue form UI (not `gh`, the API or a pull request, which are not processed), submit:
   - [Submit new Provider](https://github.com/opentofu/registry/issues/new?assignees=&labels=provider%2Csubmission&projects=&template=provider.yml&title=Provider%3A+)
   - [Submit new Provider Signing Key](https://github.com/opentofu/registry/issues/new?assignees=&labels=provider-key%2Csubmission&projects=&template=provider_key.yml&title=Provider+Key%3A+)

   The OpenTofu team reviews and approves each submission. Published versions are
   treated as immutable, so get a release right before tagging it (see its
   [POLICY.md](https://github.com/opentofu/registry/blob/main/POLICY.md)).

## Release checklist

1. Merge any pending Wayfinder bump pull request first, so the release contains the newest
   Wayfinder release you intend to support.
2. On `main`, confirm:
   - CI is green (`test.yml`, `manifest.yml`).
   - Generated code is current: `go generate ./... && git diff --exit-code`.
   - Docs are regenerated: `make docs && git diff --exit-code`.
   - `schema/wayfinder-manifest.json` has the expected `wayfinderVersion` and
     `wayfinderCommit` (they match `WAYFINDER_REF`).
3. Update [CHANGELOG.md](CHANGELOG.md): rename `Unreleased` to the new version with the date
   and add the compatibility line (below). Open a pull request and merge it.
4. Dry run locally (no signing, no publishing, nothing is uploaded):

   ```sh
   GORELEASER_CURRENT_TAG=vX.Y.Z goreleaser release --snapshot --clean --skip=sign,publish
   ```

   This should produce zip archives for linux, darwin, windows and freebsd in `dist/`
   (`dist/` is gitignored).
5. Tag the merge commit and push the tag:

   ```sh
   git tag -s vX.Y.Z -m "vX.Y.Z"   # or: git tag vX.Y.Z
   git push origin vX.Y.Z
   ```

   Use semantic versioning with a leading `v`: patch for fixes, minor for new resources or
   attributes (including a Wayfinder bump that only adds), major for breaking changes
   (removed or renamed resources or attributes, including a Wayfinder bump that removes
   them). The registries only accept `vX.Y.Z` tags.
6. Watch the *Release* workflow. It builds, checksums (`_SHA256SUMS`), signs the checksum
   file (`_SHA256SUMS.sig`) and publishes the GitHub release together with
   `_manifest.json` (from `terraform-registry-manifest.json`).

## Compatibility statement

Provider `vX.Y.Z` is built from one Wayfinder release. Record it in two places:

- `schema/wayfinder-manifest.json` (`wayfinderVersion`, `wayfinderCommit`), updated by the
  bump workflow.
- The CHANGELOG entry for the release: `Built from Wayfinder release <tag>.`

## Verify a release

1. Confirm the GitHub release has the platform zips, `_SHA256SUMS`, `_SHA256SUMS.sig` and
   `_manifest.json`.
2. Verify the signature:

   ```sh
   gpg --verify terraform-provider-wayfinder_X.Y.Z_SHA256SUMS.sig terraform-provider-wayfinder_X.Y.Z_SHA256SUMS
   sha256sum -c --ignore-missing terraform-provider-wayfinder_X.Y.Z_SHA256SUMS
   ```
3. Check that the Terraform Registry lists the version (it can take a few minutes), then
   in an empty directory with `source = "appvia/wayfinder"` and `version = "X.Y.Z"`:

   ```sh
   terraform init
   ```

   `terraform init` verifies the signature against the registered key and fails if it does
   not match. Do the same with `tofu init` once the OpenTofu registry has synced.
