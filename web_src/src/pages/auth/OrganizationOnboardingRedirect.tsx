import { useAccount } from "@/contexts/useAccount";
import { useEffect, useRef, useState } from "react";

type ProvisionedWorkspace = {
  organizationSlug: string;
  workspaceKey: string;
};

/** Provisions an organization, then opens the existing factory setup wizard. */
export function OrganizationOnboardingRedirect() {
  const { account } = useAccount();
  const hasStartedProvisioning = useRef(false);
  const [error, setError] = useState<string | null>(null);
  const githubAccount = account?.linked_accounts?.find((candidate) => candidate.provider === "github");
  const githubProvider = account?.providers?.find((candidate) => candidate.provider === "github");
  const owner = githubAccount?.name || githubAccount?.username || githubProvider?.username || "";
  const onboardingAttemptID = useRef(getOnboardingAttemptID());

  useEffect(() => {
    if (!account || hasStartedProvisioning.current) return;

    if (!owner) {
      window.location.replace("/auth/github?redirect=/onboarding");
      return;
    }

    hasStartedProvisioning.current = true;
    void provisionWorkspace(owner, onboardingAttemptID.current).catch((provisioningError: unknown) => {
      setError(provisioningError instanceof Error ? provisioningError.message : "Could not start workspace setup.");
    });
  }, [account, owner]);

  if (!error) return null;

  return (
    <main className="flex min-h-screen items-center justify-center bg-background px-6 text-foreground">
      <p className="text-sm text-destructive">{error}</p>
    </main>
  );
}

function getOnboardingAttemptID() {
  const searchParams = new URLSearchParams(window.location.search);
  const attemptID = searchParams.get("attempt");
  if (attemptID) return attemptID;

  const createdAttemptID = crypto.randomUUID();
  searchParams.set("attempt", createdAttemptID);
  window.history.replaceState(null, "", `${window.location.pathname}?${searchParams}`);
  return createdAttemptID;
}

async function provisionWorkspace(owner: string, attemptID: string) {
  const response = await fetch("/account/onboarding", {
    method: "POST",
    credentials: "include",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ owner, attemptID }),
  });
  if (!response.ok) {
    throw new Error(await response.text());
  }

  const result = (await response.json()) as ProvisionedWorkspace;
  window.location.replace(`/${result.organizationSlug}/workspaces/${result.workspaceKey}/setup`);
}
