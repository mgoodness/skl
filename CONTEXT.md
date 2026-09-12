# skl

A CLI for installing Agent Skills (SKILL.md-based capabilities) across multiple coding-agent clients, with complete separation between global and project-scoped installs.

## Language

**Skill**:
A SKILL.md-based capability unit that can be installed — e.g. `tdd`, `web-design-guidelines`. The thing `skl add` fetches and copies.

**Adapter**:
A consuming coding-agent client that skl installs skills for: `claude-code`, `kit`, or `universal`. Each adapter has its own independent destination directory, chosen so a skill can be installed for one adapter without also reaching another; see ADR-0002.
_Avoid_: Tool, agent (as a synonym for adapter) — "agent" is reserved for the AI coding agent product itself (Claude Code, Kit), and "tool" is ambiguous with a skill (colloquially "a tool for the job") and with Kit's own `RegisterTool` concept.

**Universal**:
The adapter targeting the `.agents/skills/` cross-client convention, the location the Agent Skills spec recommends and several clients (including Kit) scan by default. It is an independent destination like `claude-code` and `kit`, not a shared store those adapters write through; see ADR-0002.

**Destination**:
The concrete directory a skill is copied into for a given adapter and scope, e.g. `.claude/skills/tdd` or `~/.config/kit/skills/tdd`.

**Source**:
Where `skl add` fetches a skill from: a GitHub shorthand, full GitHub URL, GitHub tree-path, or local filesystem path. May contain one skill or many.

**Group**:
A named, selectable set of skills within a source, used as a value for `--skill` alongside individual skill names. Two kinds: a **directory group** (skills sharing a parent path within the source, e.g. `skills/engineering`) and a **plugin group** (skills declared in a plugin manifest — a `.claude-plugin/plugin.json`'s own `skills[]`, or a `.claude-plugin/marketplace.json` `plugins[]` entry's `skills[]` — discovered via Plugin Manifest Discovery, addressed by the plugin's name). A skill may belong to one directory group and, optionally, one plugin group at the same time.

**Lockfile**:
The single record of what skl has installed, per scope. There is no separate desired-state manifest in v1.0 (unlike, say, `package.json` vs. `package-lock.json`) — the lockfile is both the record and, in future versions, the thing `update`/`init` would restore from.
_Avoid_: Manifest (as a synonym) — reserve "manifest" for `.claude-plugin/marketplace.json` / `plugin.json`, which are a different kind of file entirely (declared by a source, not owned by skl).

**Expand**:
The outcome when `skl add` targets a skill whose name and source already match an existing lockfile entry: the entry's `adapters` map gains the newly requested adapter(s) rather than a new entry being created. See ADR-0003 for why identity is matched by source, not name alone.

**Conflict**:
The outcome when `skl add` targets a skill whose name matches an existing lockfile entry but whose source differs. Refused unless `--force` is given.
