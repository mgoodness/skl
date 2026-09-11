# Copy-only installation, no symlinks

skl installs a skill by copying it into each targeted adapter's destination directory. It never symlinks. This diverges from `vercel-labs/skills`, `degit`, and similar tools, which default to symlinking a canonical copy into each destination for "single source of truth, easy updates."

We chose copy-only because symlinks reintroduce exactly the problem `vercel-labs/skills` hit in its own issue tracker (issue #1199): symlinks don't survive cross-platform git checkouts reliably, and the canonical intermediate directory they point at isn't something a team wants to commit. Copy-only also removes an entire class of complexity this project would otherwise need to build: Windows junction handling, a symlink-failure-falls-back-to-copy path, and a persistent "stash" directory to symlink from and prune.

The trade-off is real: disk usage triples when a skill is installed to all three adapters, and there's no single edit point — updating a skill later means re-running install for every adapter it's copied to, not editing one canonical file.

## Considered Options

- **Symlink with copy fallback** (upstream's approach): rejected for the cross-platform/git-portability problems above, and for the added implementation surface (junctions, stash pruning, gitignore guidance) it would have required.
