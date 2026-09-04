import { useAccount } from "@/contexts/useAccount";
import { useEffect, useRef, useState, type ReactNode } from "react";

import { organizationNameFromAccount } from "./organizationNameFromAccount";

export type ProvisionedWorkspace = {
  organizationSlug: string;
  workspaceKey: string;
};

interface OrganizationOnboardingRedirectProps {
  renderWorkspace: (workspace: ProvisionedWorkspace, entryPath: string) => ReactNode;
}

/** Provisions the internal workspace and renders its existing setup wizard at /onboarding. */
export function OrganizationOnboardingRedirect({ renderWorkspace }: OrganizationOnboardingRedirectProps) {
  const { account } = useAccount();
  const hasStartedProvisioning = useRef(false);
  const [error, setError] = useState<string | null>(null);
  const [workspace, setWorkspace] = useState<ProvisionedWorkspace | null>(null);
  const owner = organizationNameFromAccount(account);
  const onboardingAttempt = useRef(getOnboardingAttempt());

  useEffect(() => {
    if (!account || hasStartedProvisioning.current || !owner) return;

    hasStartedProvisioning.current = true;
    void provisionWorkspace(owner, onboardingAttempt.current.id)
      .then(setWorkspace)
      .catch((provisioningError: unknown) => {
        setError(provisioningError instanceof Error ? provisioningError.message : "Could not start workspace setup.");
      });
  }, [account, owner]);

  if (workspace) {
    return renderWorkspace(workspace, onboardingAttempt.current.entryPath);
  }

  if (!error) return null;

  return (
    <main className="flex min-h-screen items-center justify-center bg-background px-6 text-foreground">
      <p className="text-sm text-destructive">{error}</p>
    </main>
  );
}

function getOnboardingAttempt(): { id: string; entryPath: string } {
  const searchParams = new URLSearchParams(window.location.search);
  searchParams.delete("auth_error");
  searchParams.delete("auth_link_result");
  searchParams.delete("linked_account");
  searchParams.delete("provider");
  let attemptID = searchParams.get("attempt");

  if (!attemptID) {
    attemptID = crypto.randomUUID();
    searchParams.set("attempt", attemptID);
  }

  window.history.replaceState(null, "", `${window.location.pathname}?${searchParams}`);

  return {
    id: attemptID,
    entryPath: `${window.location.pathname}?${searchParams}`,
  };
}

async function provisionWorkspace(owner: string, attemptID: string): Promise<ProvisionedWorkspace> {
  const response = await fetch("/account/onboarding", {
    method: "POST",
    credentials: "include",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ owner, attemptID }),
  });
  if (!response.ok) {
    throw new Error(await response.text());
  }

  return (await response.json()) as ProvisionedWorkspace;
}
