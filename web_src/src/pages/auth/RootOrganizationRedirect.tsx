import { useAccount } from "@/contexts/useAccount";
import { useAccountOrganizations } from "@/hooks/useAccountOrganizations";
import { useExperimentalFeature } from "@/hooks/useExperimentalFeature";
import { useLastLocation } from "@/hooks/useLastLocation";
import { organizationMatchesRoute, organizationRouteId } from "@/lib/accountOrganizations";
import { FEATURE_FACTORIES } from "@/lib/experimentalFeatures";
import { readLastVisitedLocation } from "@/lib/lastVisitedLocation";
import { pickAutoRedirectOrganization, readLastVisitedOrganization } from "@/lib/lastVisitedOrganization";
import { Navigate } from "react-router";

function LoadingView() {
  return <div className="flex min-h-screen items-center justify-center text-sm text-muted-foreground">Loading...</div>;
}

function resolveResumePath(
  accountId: string,
  routeId: string,
  lastLocation: { data?: string | null; isError: boolean },
): string | null {
  if (lastLocation.isError) {
    return readLastVisitedLocation(accountId, routeId);
  }

  return lastLocation.data ?? null;
}

function routeIdOrNull(organization: { id: string; slug?: string } | undefined): string | null {
  return organization ? organizationRouteId(organization) : null;
}

export function RootOrganizationRedirect() {
  const { account } = useAccount();
  const organizations = useAccountOrganizations();
  const organizationRoute = pickAutoRedirectOrganization(
    organizations.data?.map((organization) => ({ slug: organizationRouteId(organization) })) ?? [],
    account ? readLastVisitedOrganization(account.id) : null,
  );
  const organization = organizations.data?.find((candidate) =>
    organizationRoute ? organizationMatchesRoute(candidate, organizationRoute) : false,
  );
  const factories = useExperimentalFeature(organization?.id);
  const routeId = routeIdOrNull(organization);
  // "Resume where you left off": if this account has a saved screen for the
  // organization they are about to land in (e.g. a pending approval), go
  // there instead of the organization's default page. The backend is the
  // source of truth; local storage only covers this browser when the
  // request fails (offline, brief outage).
  const lastLocation = useLastLocation(routeId);

  if (organizations.isLoading || !account) return <LoadingView />;

  if (organizations.isError) {
    return (
      <div className="flex min-h-screen items-center justify-center px-6 text-center text-sm text-muted-foreground">
        We could not load your organizations. Refresh the page and try again.
      </div>
    );
  }

  if (!routeId) return <Navigate to="/onboarding" replace />;
  if (factories.isLoading || lastLocation.isLoading) return <LoadingView />;

  const resumePath = resolveResumePath(account.id, routeId, lastLocation);
  if (resumePath) return <Navigate to={resumePath} replace />;

  return <Navigate to={factories.has(FEATURE_FACTORIES) ? `/${routeId}/workspaces` : `/${routeId}`} replace />;
}
