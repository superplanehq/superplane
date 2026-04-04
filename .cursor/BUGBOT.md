# SuperPlane — Bugbot (repository-wide)

SuperPlane is a Go backend (`pkg/`, `cmd/`) and a React/Vite frontend in
`web_src/`.

User-facing product name is **SuperPlane** (capital P).

- PR titles should follow Conventional Commits with a release-type prefix (`feat:`, `fix:`, `chore:`, `docs:`);
- CI enforces this — do not duplicate that as a Bug unless the PR title is wrong and CI has not run yet.
- Do not nitpick formatting that `make format.js` / `make format.go` will fix
- focus on logic, security, API/contract correctness, and maintainability.
- Frontend-specific review expectations live in `web_src/.cursor/BUGBOT.md` and apply when pull requests touch files under `web_src/`.
