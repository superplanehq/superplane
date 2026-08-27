import { Navigate, useLocation, useParams } from "react-router";

import { RequirePermission } from "@/components/PermissionGate";
import { IntegrationDetailsRoute } from "@/pages/organization/settings/components/IntegrationDetailsRoute";
import { IntegrationSetup } from "@/pages/organization/settings/components/IntegrationSetup";
import { IntegrationSetupReturn } from "@/pages/organization/settings/components/IntegrationSetupReturn";

export function PreserveStateNavigate({ to }: { to: string }) {
  const location = useLocation();
  return <Navigate to={to} replace state={location.state} />;
}

export function OrganizationIntegrationSetupPage() {
  const { organizationId = "" } = useParams<{ organizationId: string }>();
  return <IntegrationSetup organizationId={organizationId} />;
}

export function OrganizationIntegrationDetailsPage() {
  const { organizationId = "" } = useParams<{ organizationId: string }>();
  return (
    <IntegrationSetupReturn organizationId={organizationId}>
      <RequirePermission resource="integrations" action="read">
        <IntegrationDetailsRoute organizationId={organizationId} />
      </RequirePermission>
    </IntegrationSetupReturn>
  );
}
