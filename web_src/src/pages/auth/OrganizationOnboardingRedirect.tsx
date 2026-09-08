import { useAccount } from "@/contexts/useAccount";
import { meKeys } from "@/hooks/useMe";
import { useQueryClient, type QueryClient, type QueryKey } from "@tanstack/react-query";
import { useCallback, useEffect, useRef, useState, type ReactNode } from "react";

import type { OnboardingWorkspaceResolution } from "../factories/pages/onboarding/onboardingWorkspaceResolutionContext";
import { organizationNameFromAccount } from "./organizationNameFromAccount";

export type ProvisionedWorkspace = {
  organizationSlug: string;
  workspaceKey: string;
};

function isWorkspaceResolutionQuery(queryKey: readonly unknown[], organizationSlug: string): boolean {
  return queryKey[0] === "factories" && queryKey[1] === organizationSlug && queryKey.length <= 3;
}

function workspaceResolutionQueries(
  queryClient: QueryClient,
  currentSlug: string,
  nextSlug: string,
): [QueryKey, unknown][] {
  const queries = queryClient.getQueriesData({
    predicate: (query) => isWorkspaceResolutionQuery(query.queryKey, nextSlug),
  });
  const currentUser = queryClient.getQueryData(meKeys.me(currentSlug));
  if (currentUser !== undefined) queries.push([meKeys.me(nextSlug), currentUser]);
  return queries;
}

interface OrganizationOnboardingRedirectProps {
  renderWorkspace: (
    workspace: ProvisionedWorkspace,
    entryPath: string,
    reresolveWorkspace: OnboardingWorkspaceResolution,
  ) => ReactNode;
}

/** Provisions the internal workspace and renders its existing setup wizard at /onboarding. */
export function OrganizationOnboardingRedirect({ renderWorkspace }: OrganizationOnboardingRedirectProps) {
  const { account } = useAccount();
  const queryClient = useQueryClient();
  const hasStartedProvisioning = useRef(false);
  const [error, setError] = useState<string | null>(null);
  const [workspace, setWorkspace] = useState<ProvisionedWorkspace | null>(null);
  const workspaceRef = useRef<ProvisionedWorkspace | null>(null);
  const owner = organizationNameFromAccount(account);
  const onboardingAttempt = useRef(getOnboardingAttempt());

  // A new organization can receive the slug of an earlier onboarding
  // organization that was later renamed. Clear that organization's cached
  // data, but retain the seeded factory data and current permissions. These
  // queries keep the repository step mounted while the new slug resolves;
  // all other organization data must load again.
  const adoptWorkspace = useCallback(
    (provisioned: ProvisionedWorkspace) => {
      if (workspaceRef.current?.organizationSlug !== provisioned.organizationSlug) {
        const resolutionQueries = workspaceRef.current
          ? workspaceResolutionQueries(queryClient, workspaceRef.current.organizationSlug, provisioned.organizationSlug)
          : [];
        queryClient.removeQueries({
          predicate: (query) => query.queryKey.includes(provisioned.organizationSlug),
        });
        for (const [queryKey, data] of resolutionQueries) {
          if (data !== undefined) queryClient.setQueryData(queryKey, data);
        }
      }
      workspaceRef.current = provisioned;
      setWorkspace(provisioned);
    },
    [queryClient],
  );

  useEffect(() => {
    if (!account || hasStartedProvisioning.current) return;
    if (!owner) {
      setError("Could not start workspace setup. Add a name to your SuperPlane account and try again.");
      return;
    }

    hasStartedProvisioning.current = true;
    void provisionWorkspace(owner, onboardingAttempt.current.id)
      .then(adoptWorkspace)
      .catch((provisioningError: unknown) => {
        setError(provisioningError instanceof Error ? provisioningError.message : "Could not start workspace setup.");
      });
  }, [account, owner, adoptWorkspace]);

  // Re-runs the retry-safe onboarding endpoint for the same attempt, so a
  // workspace can move to a new organization slug (for example, after the
  // organization is renamed from the GitHub owner) without a page reload.
  const reresolveWorkspace = useCallback(async () => {
    if (!owner) return;
    const result = await provisionWorkspace(owner, onboardingAttempt.current.id);
    adoptWorkspace(result);
  }, [owner, adoptWorkspace]);

  if (workspace) {
    return renderWorkspace(workspace, onboardingAttempt.current.entryPath, reresolveWorkspace);
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
