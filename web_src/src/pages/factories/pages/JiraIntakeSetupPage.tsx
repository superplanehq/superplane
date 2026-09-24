import { usePermissions } from "@/contexts/usePermissions";
import { usePageTitle } from "@/hooks/usePageTitle";
import { Navigate, useNavigate, useParams, useSearchParams } from "react-router";

import { useFactoriesLayout, type FactoriesLayoutContextValue } from "../layout/factoriesLayoutContext";
import {
  factoryHomePath,
  factoryLineDetailPath,
  firstFactoryLineId,
  jiraIntakeIntegrationIdFromSearch,
} from "../lib/factoryPagePaths";
import { JiraIntakeSetupDialog } from "./JiraIntakeSetupDialog";
import { JIRA_INTAKE_SETUP_COPY } from "./jiraIntakeSetupCopy";

export function JiraIntakeSetupPage() {
  const layout = useFactoriesLayout();
  const { canAct } = usePermissions();
  const { lineId } = useParams<{ lineId?: string }>();
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const model = resolveJiraIntakeSetupModel(layout, canAct("factories", "update"), lineId);

  usePageTitle(model.titleParts);

  if (!model.dialog) {
    return <Navigate to={model.redirectTo} replace />;
  }

  return (
    <div className="h-full min-h-0" data-testid="jira-intake-setup-page">
      <JiraIntakeSetupDialog
        organizationId={model.dialog.organizationId}
        factoryId={model.dialog.factoryId}
        selectIntegrationId={jiraIntakeIntegrationIdFromSearch(searchParams.toString())}
        onClose={() => navigate(model.returnHref)}
        onCreated={() => navigate(model.returnHref)}
      />
    </div>
  );
}

function resolveJiraIntakeSetupModel(
  layout: FactoriesLayoutContextValue,
  canUpdate: boolean,
  lineId: string | undefined,
): {
  titleParts: string[];
  redirectTo: string;
  returnHref: string;
  dialog?: { organizationId: string; factoryId: string };
} {
  const { organizationId, factoryId, factoryKey, factory } = layout;
  const workspaceName = factory?.name ?? "Workspace";
  const lines = factory?.lines ?? [];
  const boardHref = factoryHomePath(organizationId, factoryKey, firstFactoryLineId(factory));
  const line = lines.find((entry) => entry.id === lineId);
  const returnHref = line?.id ? factoryLineDetailPath(organizationId, factoryKey, line.id) : boardHref;
  const titleParts = [JIRA_INTAKE_SETUP_COPY.pageTitle, workspaceName];

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
