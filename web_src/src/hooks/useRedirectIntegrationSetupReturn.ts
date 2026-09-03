import { useEffect } from "react";
import { useLocation, useNavigate } from "react-router";

import {
  consumeIntegrationSetupReturn,
  hasIntegrationSetupStay,
  peekIntegrationSetupReturn,
} from "@/lib/integrationSetupReturn";

function isLegacyIntegrationDetailsPath(organizationId: string, pathname: string): boolean {
  const segments = pathname.split("/").filter(Boolean);
  return (
    segments.length === 4 &&
    segments[0] === organizationId &&
    segments[1] === "settings" &&
    segments[2] === "integrations"
  );
}

/**
 * Returns a provider callback to its initiating page before legacy settings
 * redirects can replace the integration details route.
 */
export function useRedirectIntegrationSetupReturn(organizationId: string | undefined): void {
  const location = useLocation();
  const navigate = useNavigate();

  useEffect(() => {
    if (!organizationId || !isLegacyIntegrationDetailsPath(organizationId, location.pathname)) return;
    if (hasIntegrationSetupStay(location.search)) return;

    const returnTo = peekIntegrationSetupReturn(organizationId);
    if (!returnTo) return;

    // The provider can finish on a later callback after setup has already
    // completed. Consume this one-shot return before navigating so that late
    // callbacks cannot reopen the setup wizard.
    consumeIntegrationSetupReturn(organizationId);
    navigate(returnTo, { replace: true });
  }, [location.pathname, location.search, navigate, organizationId]);
}
