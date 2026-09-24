import { useAccount } from "@/contexts/useAccount";
import { useAccountOrganizations } from "@/hooks/useAccountOrganizations";
import { useExperimentalFeature } from "@/hooks/useExperimentalFeature";
import { useFactories } from "@/hooks/useFactoryData";
import { useLastLocation } from "@/hooks/useLastLocation";
import { organizationMatchesRoute, organizationRouteId } from "@/lib/accountOrganizations";
import { FEATURE_FACTORIES } from "@/lib/experimentalFeatures";
import { readLastVisitedLocation } from "@/lib/lastVisitedLocation";
import {
  pickAutoRedirectOrganization,
  pickResumePath,
  readLastVisitedOrganization,
} from "@/lib/lastVisitedOrganization";
import { Navigate } from "react-router";

import { incompleteWorkspaceSetupPath } from "../factories/pages/onboarding/onboardingResumePath";

type AccountOrganization = NonNullable<ReturnType<typeof useAccountOrganizations>["data"]>[number];
type Account = NonNullable<ReturnType<typeof useAccount>["account"]>;

function LoadingView() {
  return <div className="flex min-h-screen items-center justify-center text-sm text-muted-foreground">Loading...</div>;
}

function resolveResumePath(
  accountId: string,
  routeId: string,
  lastLocation: { data?: string | null; isError: boolean },
  listPath?: string,
): string | null {
  const localPath = readLastVisitedLocation(accountId, routeId);
  if (lastLocation.isError) {
    return pickResumePath(routeId, listPath, localPath);
  }

  return pickResumePath(routeId, lastLocation.data, listPath, localPath);
}

function isWaitingForRedirect(args: {
  experimentalLoading: boolean;
  lastLocationLoading: boolean;
  canLoadWorkspaces: boolean;
  workspacesLoading: boolean;
}): boolean {
  if (args.experimentalLoading || args.lastLocationLoading) {
    return true;
  }
  return args.canLoadWorkspaces && args.workspacesLoading;
}

function pickRootRedirectPath(args: {
  accountId: string;
  routeId: string;
  lastLocation: { data?: string | null; isError: boolean };
  listPath?: string;
  factoriesEnabled: boolean;
  workspaces?: Parameters<typeof incompleteWorkspaceSetupPath>[1];
}): string {
  return (
    resolveResumePath(args.accountId, args.routeId, args.lastLocation, args.listPath) ??
    incompleteWorkspaceSetupPath(args.routeId, args.workspaces) ??
    (args.factoriesEnabled ? `/${args.routeId}/workspaces` : `/${args.routeId}`)
  );
}

function selectedOrganization(
  organizations: AccountOrganization[] | undefined,
  account: Account | null | undefined,
): AccountOrganization | undefined {
  const organizationRoute = pickAutoRedirectOrganization(
    organizations?.map((organization) => ({
      slug: organizationRouteId(organization),
      lastLocationUpdatedAt: organization.lastLocationUpdatedAt,
    })) ?? [],
    account ? readLastVisitedOrganization(account.id) : null,
  );
  if (!organizationRoute || !organizations) {
    return undefined;
  }
  return organizations.find((candidate) => organizationMatchesRoute(candidate, organizationRoute));
}

function classifyRootRedirect(args: {
  account: Account | null | undefined;
  organizationsLoading: boolean;
  organizationsError: boolean;
  routeId: string | null;
  waiting: boolean;
  lastLocation: { data?: string | null; isError: boolean };
  listPath?: string;
  factoriesEnabled: boolean;
  workspaces?: Parameters<typeof incompleteWorkspaceSetupPath>[1];
}): { kind: "loading" } | { kind: "error" } | { kind: "ready"; path: string } {
  if (args.organizationsLoading || !args.account) return { kind: "loading" };
  if (args.organizationsError) return { kind: "error" };
  if (!args.routeId) return { kind: "ready", path: "/onboarding" };
  if (args.waiting) return { kind: "loading" };
  return {
    kind: "ready",
    path: pickRootRedirectPath({
      accountId: args.account.id,
      routeId: args.routeId,
      lastLocation: args.lastLocation,
      listPath: args.listPath,
      factoriesEnabled: args.factoriesEnabled,
      workspaces: args.workspaces,
    }),
  };
}

function useRootRedirectPath(): ReturnType<typeof classifyRootRedirect> {
  const { account } = useAccount();
  const organizations = useAccountOrganizations();
  const organization = selectedOrganization(organizations.data, account);
  const experimentalFeatures = useExperimentalFeature(organization?.id);
  const routeId = organization ? organizationRouteId(organization) : null;
  const lastLocation = useLastLocation(routeId);
  const canLoadWorkspaces = Boolean(organization?.id && experimentalFeatures.has(FEATURE_FACTORIES));
  const workspaces = useFactories(organization?.id ?? "", canLoadWorkspaces);

  return classifyRootRedirect({
    account,
    organizationsLoading: organizations.isLoading,
    organizationsError: organizations.isError,
    routeId,
    waiting: isWaitingForRedirect({
      experimentalLoading: experimentalFeatures.isLoading,
      lastLocationLoading: lastLocation.isLoading,
      canLoadWorkspaces,
      workspacesLoading: workspaces.isLoading,
    }),
    lastLocation,
    listPath: organization?.lastLocationPath,
    factoriesEnabled: experimentalFeatures.has(FEATURE_FACTORIES),
    workspaces: workspaces.data,
  });
}

export function RootOrganizationRedirect() {
  const redirect = useRootRedirectPath();

  if (redirect.kind === "loading") return <LoadingView />;
  if (redirect.kind === "error") {
    return (
      <div className="flex min-h-screen items-center justify-center px-6 text-center text-sm text-muted-foreground">
        We could not load your organizations. Refresh the page and try again.
      </div>
    );
  }

  return <Navigate to={redirect.path} replace />;
}
