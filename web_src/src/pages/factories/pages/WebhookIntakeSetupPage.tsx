import { usePermissions } from "@/contexts/usePermissions";
import { usePageTitle } from "@/hooks/usePageTitle";
import { Navigate, useNavigate, useParams } from "react-router";

import { useFactoriesLayout } from "../layout/factoriesLayoutContext";
import { factoryHomePath, factoryLineDetailPath, firstFactoryLineId } from "../lib/factoryPagePaths";
import { WebhookIntakeSetupDialog } from "./WebhookIntakeSetupDialog";
import { WEBHOOK_INTAKE_SETUP_COPY } from "./webhookIntakeSetupCopy";

export function WebhookIntakeSetupPage() {
  const { organizationId, factoryId, routeSegment, factory } = useFactoriesLayout();
  const { canAct } = usePermissions();
  const { lineId } = useParams<{ lineId?: string }>();
  const navigate = useNavigate();
  const line = factory?.lines?.find((entry) => entry.id === lineId);
  const boardHref = factoryHomePath(organizationId, routeSegment, firstFactoryLineId(factory));
  const returnHref = line?.id ? factoryLineDetailPath(organizationId, routeSegment, line.id) : boardHref;

  usePageTitle([WEBHOOK_INTAKE_SETUP_COPY.pageTitle, factory?.name ?? "Workspace"]);

  if (!canAct("factories", "update") || !lineId || (factory && !line)) {
    return <Navigate to={boardHref} replace />;
  }

  return (
    <div className="h-full min-h-0" data-testid="webhook-intake-setup-page">
      <WebhookIntakeSetupDialog
        organizationId={organizationId}
        factoryId={factoryId}
        factoryKey={routeSegment}
        onClose={() => navigate(returnHref)}
        onCreated={() => navigate(returnHref)}
      />
    </div>
  );
}
