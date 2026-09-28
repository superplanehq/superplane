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
import { pathBelongsToOrganization } from "@/lib/safeRedirectPath";
import { Navigate } from "react-router";

import { factoryDetailPath, factoryListPath, factorySetupPath } from "../factories/lib/factoryPagePaths";
import { pickReadyFactory, readLastVisitedFactory } from "../factories/lib/lastVisitedFactory";

type AccountOrganization = NonNullable<ReturnType<typeof useAccountOrganizations>["data"]>[number];
type Account = NonNullable<ReturnType<typeof useAccount>["account"]>;
type WorkspaceHomeCandidate = {
  id?: string;
  key?: string;
  onboarding?: { completedAt?: string };
};
type SavedScreen = {
  accountId: string;
  routeId: string;
  lastLocation: { data?: string | null; isError: boolean };
  listPath?: string;
};

function LoadingView() {
  return <div className="flex min-h-screen items-center justify-center text-sm text-muted-foreground">Loading...</div>;
}

function resolveResumePath(screen: SavedScreen): string | null {
  return pickResumePath(screen.routeId, ...savedScreenCandidates(screen));
}

function savedScreenCandidates(screen: SavedScreen): Array<string | null | undefined> {
  const localPath = readLastVisitedLocation(screen.accountId, screen.routeId);
  if (screen.lastLocation.isError) {
    return [screen.listPath, localPath];
  }
  return [screen.lastLocation.data, screen.listPath, localPath];
}

function workspaceKeyFromPath(path: string, routeId: string): string | null {
  const pathname = path.split(/[?#]/, 1)[0] ?? "";
  const prefix = `/${routeId}/workspaces/`;
  if (!pathname.startsWith(prefix)) {
    return null;
  }

  const segment = pathname.slice(prefix.length).split("/")[0] ?? "";
  if (!segment || segment.toLowerCase() === "new") {
    return null;
  }
  return segment;
}

function workspaceIdForKey(workspaces: WorkspaceHomeCandidate[], key: string): string | null {
  const match = workspaces.find(
    (workspace) => workspace.id && workspace.key && workspace.key.toLowerCase() === key.toLowerCase(),
  );
  return match?.id ?? null;
}

function workspaceIdFromSavedScreen(screen: SavedScreen, workspaces: WorkspaceHomeCandidate[]): string | null {
  for (const candidate of savedScreenCandidates(screen)) {
    if (!candidate || !pathBelongsToOrganization(candidate, screen.routeId)) {
      continue;
    }
    const key = workspaceKeyFromPath(candidate, screen.routeId);
    if (!key) {
      continue;
    }
    const id = workspaceIdForKey(workspaces, key);
    if (id) {
      return id;
    }
  }
  return null;
}

function preferredWorkspaceId(screen: SavedScreen, workspaces: WorkspaceHomeCandidate[]): string | null {
  const lastVisitedId = readLastVisitedFactory(screen.accountId, screen.routeId);
  if (lastVisitedId && workspaces.some((workspace) => workspace.id === lastVisitedId)) {
    return lastVisitedId;
  }

  return workspaceIdFromSavedScreen(screen, workspaces);
}

function pathForChosenWorkspace(routeId: string, workspace: WorkspaceHomeCandidate): string | null {
  if (!workspace.key) {
    return null;
  }
  if (workspace.onboarding?.completedAt) {
    return factoryDetailPath(routeId, workspace.key);
  }
  return factorySetupPath(routeId, workspace.key);
}

function pickWorkspaceHomePath(args: SavedScreen & { workspaces?: WorkspaceHomeCandidate[] }): string {
  const workspaces = args.workspaces ?? [];
  const chosen = pickReadyFactory(workspaces, preferredWorkspaceId(args, workspaces));
  return (chosen && pathForChosenWorkspace(args.routeId, chosen)) || factoryListPath(args.routeId);
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

function pickRootRedirectPath(
  args: SavedScreen & { factoriesEnabled: boolean; workspaces?: WorkspaceHomeCandidate[] },
): string {
  if (args.factoriesEnabled) {
    return pickWorkspaceHomePath(args);
  }

  return resolveResumePath(args) ?? `/${args.routeId}`;
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
  workspaces?: WorkspaceHomeCandidate[];
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
