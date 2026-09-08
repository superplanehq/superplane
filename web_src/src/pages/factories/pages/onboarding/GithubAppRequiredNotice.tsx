/**
 * Blocks workspace setup on installations that hold no SuperPlane GitHub App
 * credentials. Setup cannot connect GitHub without them, so the wizard would
 * fail midway. This state only occurs on self-managed or local installations.
 */
export function GithubAppRequiredNotice() {
  return (
    <div className="flex min-h-full items-center justify-center bg-background px-6 text-foreground">
      <div
        className="max-w-md rounded-lg border border-border bg-card p-8 text-center"
        data-testid="github-app-required"
      >
        <h1 className="text-[16px] font-semibold">Workspace setup is not available</h1>
        <p className="mt-2 text-[13px] text-muted-foreground">
          This installation has no GitHub App configured, so setup cannot connect GitHub. Set the
          SUPERPLANE_GITHUB_APP_* environment variables and restart the server. See the local GitHub App guide in
          docs/contributing.
        </p>
      </div>
    </div>
  );
}
