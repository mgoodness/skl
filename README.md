# skl

A CLI for installing Agent Skills (SKILL.md-based capabilities) across multiple coding-agent clients, with complete separation between global and project-scoped installs.

See [CONTEXT.md](CONTEXT.md) for the project's glossary and `docs/adr/` for its binding design decisions.

## Status

v1.0 is under active development, tracked as [GitHub issues](https://github.com/mgoodness/skl/issues) under #1. Currently implemented:

- `skl add <local-path>`: installs a skill from a local directory containing a root `SKILL.md` into all three adapters' project destinations (`.claude/skills/`, `.kit/skills/`, `.agents/skills/`), recording the install in `.skl-lock.json`.
- `skl add --global <local-path>`: installs into each adapter's global destination (`~/.claude/skills/`, `~/.config/kit/skills/`, `~/.agents/skills/`) instead, recording the install in the global lockfile at `$XDG_DATA_HOME/skl/lock.json` (falling back to `~/.local/share/skl/lock.json`).

## Build

```console
go build -o skl .
```

## Usage

```console
skl add ./path/to/some-skill
skl add --global ./path/to/some-skill
```

A multi-skill source (a root with a `skills/` directory containing several skills) requires `--skill` to say which to install: individual skill names, **directory-group** paths naming a shared parent, and/or **plugin-group** names, freely mixed, or `"*"` for every skill found.

```console
# every skill under the source's skills/engineering/ directory
skl add owner/repo --skill skills/engineering
# a skill name and a directory group together
skl add owner/repo --skill tdd,skills/writing
# every skill a source's .claude-plugin/plugin.json (or marketplace.json) declares
skl add owner/repo --skill my-plugin
# every skill the source contains
skl add owner/repo --skill "*"
```

When a source carries a Claude plugin manifest (`.claude-plugin/plugin.json` or `.claude-plugin/marketplace.json`) at its root, the skills it declares form a **plugin group** addressed by the plugin's name — alongside the directory groups, and installed exactly the same way (there is no separate `--plugin` flag). Each installed skill's lockfile entry records the plugin it belongs to as `pluginName`.

## Releasing

Releases are cut by [release-please](https://github.com/googleapis/release-please) from
Conventional Commits on `main`, and built by [GoReleaser](https://goreleaser.com) once
release-please tags a version. One-time setup (a GitHub App release-please's workflow
needs to push its tag) is scripted, not manual: run
[`scripts/setup-release-please-app.sh`](scripts/setup-release-please-app.sh).
