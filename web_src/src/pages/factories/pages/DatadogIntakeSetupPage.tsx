import { usePermissions } from "@/contexts/usePermissions";
import { usePageTitle } from "@/hooks/usePageTitle";
import { Navigate, useNavigate, useParams } from "react-router";

import { useFactoriesLayout, type FactoriesLayoutContextValue } from "../layout/factoriesLayoutContext";
import { factoryHomePath, factoryLineDetailPath, firstFactoryLineId } from "../lib/factoryPagePaths";
import { DatadogIntakeSetupDialog } from "./DatadogIntakeSetupDialog";
import { DATADOG_INTAKE_SETUP_COPY } from "./datadogIntakeSetupCopy";

export function DatadogIntakeSetupPage() {
  const layout = useFactoriesLayout();
  const { canAct } = usePermissions();
  const { lineId } = useParams<{ lineId?: string }>();
  const navigate = useNavigate();
  const model = resolveDatadogIntakeSetupModel(layout, canAct("factories", "update"), lineId);

  usePageTitle(model.titleParts);

  if (!model.dialog) {
    return <Navigate to={model.redirectTo} replace />;
  }

  return (
    <div className="h-full min-h-0" data-testid="datadog-intake-setup-page">
      <DatadogIntakeSetupDialog
        organizationId={model.dialog.organizationId}
        factoryId={model.dialog.factoryId}
        onClose={() => navigate(model.returnHref)}
        onCreated={() => navigate(model.returnHref)}
      />
    </div>
  );
}

function resolveDatadogIntakeSetupModel(
  layout: FactoriesLayoutContextValue,
  canUpdate: boolean,
  lineId: string | undefined,
): {
  titleParts: string[];
  redirectTo: string;
  returnHref: string;
  dialog?: { organizationId: string; factoryId: string };
} {
  const { organizationId, factoryId, routeSegment, factory } = layout;
  const workspaceName = factory?.name ?? "Workspace";
  const lines = factory?.lines ?? [];
  const boardHref = factoryHomePath(organizationId, routeSegment, firstFactoryLineId(factory));
  const line = lines.find((entry) => entry.id === lineId);
  const returnHref = line?.id ? factoryLineDetailPath(organizationId, routeSegment, line.id) : boardHref;
  const titleParts = [DATADOG_INTAKE_SETUP_COPY.pageTitle, workspaceName];

  if (!canUpdate || !lineId || (factory && !line)) {
    return { titleParts, redirectTo: boardHref, returnHref };
  }

  return {
    titleParts,
    redirectTo: "",
    returnHref,
    dialog: { organizationId, factoryId },
  };
}
