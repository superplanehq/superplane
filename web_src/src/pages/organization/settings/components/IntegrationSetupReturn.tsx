import { Loader2 } from "lucide-react";
import type { ReactNode } from "react";
import { useEffect, useState } from "react";
import { useNavigate } from "react-router";

import { consumeIntegrationSetupReturn, peekIntegrationSetupReturn } from "@/lib/integrationSetupReturn";

interface IntegrationSetupReturnProps {
  organizationId: string;
  children: ReactNode;
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
  // Read once on mount. After the marker is consumed the value must stay set,
  // so the children never flash before the navigation happens.
  const [returnTo] = useState(() => peekIntegrationSetupReturn(organizationId));

  useEffect(() => {
    if (!returnTo) return;
    consumeIntegrationSetupReturn(organizationId);
    navigate(returnTo, { replace: true });
  }, [navigate, organizationId, returnTo]);

  if (returnTo) {
    return (
      <div className="flex min-h-screen items-center justify-center" data-testid="integration-setup-return">
        <Loader2 className="size-8 animate-spin text-gray-500 dark:text-gray-400" />
      </div>
    );
  }

  return <>{children}</>;
}
