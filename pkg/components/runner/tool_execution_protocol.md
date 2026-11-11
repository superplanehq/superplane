## Tool execution

- When a task needs an enabled workspace MCP server, discover its exposed tools and call a permitted tool.
- A server name in the prompt does not prove tool access. Verify access through tool discovery and invocation.
- Preserve disabled connections and tool restrictions. Do not call unrelated tools merely because they are configured.
- If discovery or invocation fails, report the server, attempted tool, and concrete access or authentication error.
- Do not print credentials or include them in artifacts.

## Browser navigation

- When a task needs UI navigation or inspection, use the installed browser tools.
- Read repository setup instructions before you start the UI. Run the preview in the repository container when required.
- Run the installed `playwright cli` on the runner host, not in the repository container.
- Check `playwright --version` and `playwright cli --help` on the runner host before reporting browser tools as unavailable.
- Poll the intended preview URL with a bounded timeout. Diagnose preview reachability separately from browser availability.
- Use `playwright cli open <url> --browser=chromium`, then `snapshot` or `find` to inspect rendered content.
- Use returned element references for navigation interactions. Record the route and a visible element that confirms arrival.
- If the intended UI is unavailable, attempt an isolated preview and state the limitation.
- Browser navigation alone does not require screenshots, video, or artifact uploads.
- If navigation fails, report the command and concrete error, such as a missing executable or unreachable URL.
- Close the browser session and stop temporary preview processes after inspection.
