import { useAccount } from "@/contexts/useAccount";
import { useAccountOrganizations } from "@/hooks/useAccountOrganizations";
import { useExperimentalFeature } from "@/hooks/useExperimentalFeature";
import { organizationMatchesRoute, organizationRouteId } from "@/lib/accountOrganizations";
import { FEATURE_FACTORIES } from "@/lib/experimentalFeatures";
import { pickAutoRedirectOrganization, readLastVisitedOrganization } from "@/lib/lastVisitedOrganization";
import { Navigate } from "react-router";

function LoadingView() {
  return <div className="flex min-h-screen items-center justify-center text-sm text-muted-foreground">Loading...</div>;
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

  if (organizations.isLoading || !account) return <LoadingView />;

  if (organizations.isError) {
    return (
      <div className="flex min-h-screen items-center justify-center px-6 text-center text-sm text-muted-foreground">
        We could not load your organizations. Refresh the page and try again.
      </div>
    );
  }

  if (!organization) return <Navigate to="/onboarding" replace />;
  if (factories.isLoading) return <LoadingView />;

  const routeId = organizationRouteId(organization);
  return <Navigate to={factories.has(FEATURE_FACTORIES) ? `/${routeId}/workspaces` : `/${routeId}`} replace />;
}
