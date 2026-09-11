# skl

A CLI for installing Agent Skills (SKILL.md-based capabilities) across multiple coding-agent clients, with complete separation between global and project-scoped installs.

See [CONTEXT.md](CONTEXT.md) for the project's glossary and `docs/adr/` for its binding design decisions.

## Status

v1.0 is under active development, tracked as [GitHub issues](https://github.com/mgoodness/skl/issues) under #1. Currently implemented:

- `skl add <local-path>`: installs a skill from a local directory containing a root `SKILL.md` into all three adapters' project destinations (`.claude/skills/`, `.kit/skills/`, `.agents/skills/`), recording the install in `.skl-lock.json`.

## Build

```console
go build -o skl .
```

## Usage

```console
skl add ./path/to/some-skill
```

## Releasing

Releases are cut by [release-please](https://github.com/googleapis/release-please) from
Conventional Commits on `main`, and built by [GoReleaser](https://goreleaser.com) once
release-please tags a version. One-time setup (a GitHub App release-please's workflow
needs to push its tag) is scripted, not manual: run
[`scripts/setup-release-please-app.sh`](scripts/setup-release-please-app.sh).
