import { Button } from "@/components/ui/button";
import { useAccount } from "@/contexts/useAccount";
import { useState } from "react";

type ProvisionedWorkspace = {
  organizationSlug: string;
  workspaceKey: string;
};

export function InitialOnboarding() {
  const { account } = useAccount();
  const [isCreating, setIsCreating] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const githubAccount = account?.linked_accounts?.find((candidate) => candidate.provider === "github");
  const githubProvider = account?.providers?.find((candidate) => candidate.provider === "github");
  const owner = githubAccount?.name || githubAccount?.username || githubProvider?.username || "";

  const createWorkspace = async () => {
    if (!owner || isCreating) return;

    setError(null);
    setIsCreating(true);
    try {
      const response = await fetch("/account/onboarding", {
        method: "POST",
        credentials: "include",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ owner }),
      });
      if (!response.ok) throw new Error(await response.text());

      const result = (await response.json()) as ProvisionedWorkspace;
      window.location.assign(`/${result.organizationSlug}/workspaces/${result.workspaceKey}/setup`);
    } catch (requestError) {
      setError(requestError instanceof Error && requestError.message ? requestError.message : "Could not create your workspace.");
    } finally {
      setIsCreating(false);
    }
  };

  return (
    <main className="flex min-h-screen items-center justify-center bg-background px-6 text-foreground">
      <section className="w-full max-w-md rounded-lg border border-border bg-card p-8">
        <h1 className="text-lg font-semibold">Set up your workspace</h1>
        <p className="mt-2 text-sm text-muted-foreground">
          Connect GitHub to create an organization from your GitHub account.
        </p>
        {owner ? (
          <div className="mt-6 space-y-3">
            <p className="text-sm">GitHub account: {owner}</p>
            {error ? <p className="text-sm text-destructive">{error}</p> : null}
            <Button type="button" onClick={createWorkspace} disabled={isCreating} data-testid="initial-onboarding-create-workspace">
              {isCreating ? "Creating workspace..." : "Create workspace"}
            </Button>
          </div>
        ) : (
          <Button className="mt-6" asChild>
            <a href="/auth/github?redirect=/onboarding">Connect GitHub</a>
          </Button>
        )}
      </section>
    </main>
  );
}
