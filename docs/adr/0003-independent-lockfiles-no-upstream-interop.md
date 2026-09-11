# Independent lockfiles, no interop with vercel-labs/skills

skl maintains its own lockfiles — `.skl-lock.json` (project) and `$XDG_DATA_HOME/skl/lock.json` (global) — and never reads or writes `vercel-labs/skills`' lockfiles (`skills-lock.json` project-side, `~/.agents/.skill-lock.json` global-side), even though both tools solve the same problem and a user may have both installed.

We considered matching upstream's lockfile paths and/or schema so the two tools could interoperate or at least avoid confusion. We rejected it: skl's entries need an `adapters` map recording which of three independent destinations (see ADR-0002) a skill was copied to, a concept upstream's schema has no room for, since upstream installs to a single shared location per skill. Matching upstream's *paths* would only create the appearance of compatibility without the substance; matching its *schema* would permanently constrain skl's shape to a model that doesn't fit its adapter-fanout design.

Collisions between the two tools (e.g. both installing a skill of the same name under `.agents/skills/`) are treated as the user's responsibility. skl does not scan for or warn about entries in upstream's lockfiles.
