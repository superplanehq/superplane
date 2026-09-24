import { usePermissions } from "@/contexts/usePermissions";
import { usePageTitle } from "@/hooks/usePageTitle";
import { Navigate, useNavigate, useParams } from "react-router";

import { useFactoriesLayout, type FactoriesLayoutContextValue } from "../layout/factoriesLayoutContext";
import { factoryHomePath, factoryLineDetailPath, firstFactoryLineId } from "../lib/factoryPagePaths";
import { PLANNING_SETTINGS_COPY } from "./planningSettingsCopy";
import { PlanningSetupDialog } from "./PlanningSetupDialog";
import { planningSetupRedirect } from "./planningSetupCaption";

export function PlanningSetupPage() {
  const layout = useFactoriesLayout();
  const { canAct } = usePermissions();
  const { lineId } = useParams<{ lineId?: string }>();
  const navigate = useNavigate();
  const model = resolvePlanningSetupModel(layout, canAct("factories", "update"), lineId);

  usePageTitle(model.titleParts);

  if (!model.dialog) {
    return <Navigate to={model.redirectTo} replace />;
  }

  return (
    <div className="h-full min-h-0" data-testid="planning-setup-page">
      <PlanningSetupDialog
        organizationId={model.dialog.organizationId}
        factoryId={model.dialog.factoryId}
        factory={model.dialog.factory}
        onClose={() => navigate(model.returnHref)}
        onFinished={() => navigate(model.returnHref)}
      />
    </div>
  );
}

function resolvePlanningSetupModel(
  layout: FactoriesLayoutContextValue,
  canUpdate: boolean,
  lineId: string | undefined,
): {
  titleParts: string[];
  redirectTo: string;
  returnHref: string;
  dialog?: {
    organizationId: string;
    factoryId: string;
    factory: FactoriesLayoutContextValue["factory"];
  };
} {
  const { organizationId, factoryId, factoryKey, factory } = layout;
  const boardHref = factoryHomePath(organizationId, factoryKey, firstFactoryLineId(factory));
  const line = factory?.lines?.find((entry) => entry.id === lineId);
  const returnHref = line?.id ? factoryLineDetailPath(organizationId, factoryKey, line.id) : boardHref;
  const titleParts = [PLANNING_SETTINGS_COPY.wizardPageTitle, factory?.name ?? "Workspace"];
  const redirectTo = planningSetupRedirect({
    canUpdate,
    lineId,
    factoryPresent: Boolean(factory),
    linePresent: Boolean(line),
    boardHref,
  });

  if (redirectTo || !factory) {
    return { titleParts, redirectTo: redirectTo ?? returnHref, returnHref };
  }

  return {
    titleParts,
    redirectTo: "",
    returnHref,
    dialog: {
      organizationId,
      factoryId,
      factory,
    },
  };
}
