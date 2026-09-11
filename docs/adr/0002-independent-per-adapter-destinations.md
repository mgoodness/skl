# Independent per-adapter destinations, no shared canonical store

Each adapter — `claude-code`, `kit`, `universal` — gets its own destination directory (`.claude/skills/<name>`, `.kit/skills/<name>`, `.agents/skills/<name>`, and their global equivalents). `universal` is not a shared store the other two write through or symlink from; it is one independent destination among three.

Kit is not harmed by overlap between `kit`'s and `universal`'s destinations: it scans both `.agents/skills/` and `.kit/skills/` (at both project and user scope) and de-duplicates by the skill's `name` field, with `.agents/skills/` taking precedence — installing the same skill to both would simply be redundant, not ambiguous, from Kit's point of view.

The reason to keep them separate is skl's own selectivity, not Kit's tolerance for duplicates. If `kit`'s destination were the same directory as `universal`'s, a user could no longer install a skill for Kit specifically without it also landing in the cross-client `.agents/skills/` convention that `universal` and every other spec-compliant client read — nor install a `universal` skill without it also being Kit's copy. Independent destinations preserve the ability to choose `--agent kit` or `--agent universal` (or both) and get exactly that outcome on disk.

## Consequences

A future contributor may look at `kit`'s destination and `universal`'s destination, notice Kit reads both anyway, and be tempted to merge them as a harmless simplification. Kit itself would keep working fine — but doing so removes the ability to install a skill for one of those two adapters without the other, which is the actual property this decision protects.
