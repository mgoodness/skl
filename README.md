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
