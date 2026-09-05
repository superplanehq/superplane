import { Loader2 } from "lucide-react";
import type { ReactNode } from "react";
import { useEffect } from "react";
import { useLocation, useNavigate, useSearchParams } from "react-router";

import { useExperimentalFeature } from "@/hooks/useExperimentalFeature";
import { FEATURE_FACTORIES } from "@/lib/experimentalFeatures";
import {
  hasIntegrationSetupStay,
  peekIntegrationSetupReturn,
  withGitHubSetupRequest,
} from "@/lib/integrationSetupReturn";

interface IntegrationSetupReturnProps {
  organizationId: string;
  children: ReactNode;
}

function isLegacySettingsIntegrationsPath(pathname: string): boolean {
  return pathname.includes("/settings/integrations/");
}

/**
 * Sends the browser back to the page that started an integration setup, for
 * example workspace onboarding. A provider such as GitHub always redirects to
 * the integration details page when setup finishes, so the return happens here.
 *
 * This wraps the route above the permission gate on purpose. The return must
 * work before permissions load, and also for a member who cannot read
 * integrations, because that member would otherwise get a "not found" page
 * instead of going back to the wizard.
 */
export function IntegrationSetupReturn({ organizationId, children }: IntegrationSetupReturnProps) {
  const navigate = useNavigate();
  const location = useLocation();
  const [searchParams] = useSearchParams();
  const stayOnPage = hasIntegrationSetupStay(searchParams.toString());
  const { has, isLoading: featuresLoading } = useExperimentalFeature(organizationId);
  // Peek on each render because provider callbacks use the organization UID.
  // OrganizationScope later replaces that UID with the slug that keys storage.
  // The destination page deletes the marker after navigation.
  const storedReturn = peekIntegrationSetupReturn(organizationId);
  const factoriesOnboardingFallback =
    !featuresLoading && has(FEATURE_FACTORIES) && !storedReturn && isLegacySettingsIntegrationsPath(location.pathname);
  const returnTo = storedReturn
    ? withGitHubSetupRequest(storedReturn, searchParams.toString())
    : factoriesOnboardingFallback
      ? withGitHubSetupRequest("/onboarding", searchParams.toString())
      : null;

  useEffect(() => {
    if (!returnTo || stayOnPage) return;
    navigate(returnTo, { replace: true });
  }, [navigate, returnTo, stayOnPage]);

  if (returnTo && !stayOnPage) {
    return (
      <div className="flex min-h-screen items-center justify-center" data-testid="integration-setup-return">
        <Loader2 className="size-8 animate-spin text-gray-500 dark:text-gray-400" />
      </div>
    );
  }

  return <>{children}</>;
}
