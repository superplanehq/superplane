- Whenever you need to run a "go" command, prefer make commands over docker compose
- To run tests: `make test PKG_TEST_PACKAGES=./path/to/package/...`
- Dont run format, build, tests, etc - unless specificly asked for
- Avoid rewriting git history as much as possible (no rebase, amend, etc unless explicitly asked)
- Always sign off commits for DCO: use -s for commit, --signoff for merge

## Dev Commands

- Start app: `make dev.start`
- Start app (foreground): `make dev.start.fg`
- Logs (app only, no rabbitmq): `make dev.logs.app`
- Expose local webhooks: `ngrok http 8000`
- Set webhook URL: `WEBHOOKS_BASE_URL=https://your-url.ngrok-free.dev` in `.env`
