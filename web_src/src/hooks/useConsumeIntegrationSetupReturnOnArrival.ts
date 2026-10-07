import { useEffect } from "react";
import { useLocation } from "react-router";

import { consumeIntegrationSetupReturnIfArrived } from "@/lib/integrationSetupReturn";

/** Clears the GitHub setup return marker once this page is the stored destination. */
export function useConsumeIntegrationSetupReturnOnArrival(organizationId: string | undefined): void {
  const location = useLocation();

  useEffect(() => {
    if (!organizationId) return;
    consumeIntegrationSetupReturnIfArrived(organizationId, location.pathname);
  }, [location.pathname, organizationId]);
}
