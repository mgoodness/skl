## Agent skills

### Issue tracker

Issues live as GitHub issues in `mgoodness/skl`, managed via the `gh` CLI. See `docs/agents/issue-tracker.md`.

### Triage labels

Default label vocabulary (`needs-triage`, `needs-info`, `ready-for-agent`, `ready-for-human`, `wontfix`). See `docs/agents/triage-labels.md`.

### Domain docs

Single-context: `CONTEXT.md` + `docs/adr/` at the repo root. See `docs/agents/domain.md`.

### Git workflow

Work happens on a branch, never committed directly to `main`: create a branch, push it to `origin`, and open a PR (`gh pr create`). Never merge the PR automatically — only merge when explicitly asked to.
