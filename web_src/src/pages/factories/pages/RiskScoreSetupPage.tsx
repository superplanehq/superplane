import { usePermissions } from "@/contexts/usePermissions";
import { usePageTitle } from "@/hooks/usePageTitle";
import { Navigate, useNavigate, useParams } from "react-router";

import { useFactoriesLayout } from "../layout/factoriesLayoutContext";
import { factoryHomePath, factoryLineDetailPath, firstFactoryLineId } from "../lib/factoryPagePaths";
import { RiskScoreSetupDialog } from "./RiskScoreSetupDialog";
import { RISK_SCORE_SETUP_COPY } from "./riskScoreSetupCopy";

export function RiskScoreSetupPage() {
  const { organizationId, factoryId, factoryKey, factory } = useFactoriesLayout();
  const { canAct } = usePermissions();
  const { lineId } = useParams<{ lineId?: string }>();
  const navigate = useNavigate();

  usePageTitle([RISK_SCORE_SETUP_COPY.pageTitle, factory?.name ?? "Workspace"]);

  const boardHref = factoryHomePath(organizationId, factoryKey, firstFactoryLineId(factory));
  const line = factory?.lines?.find((entry) => entry.id === lineId);
  const returnHref = line?.id ? factoryLineDetailPath(organizationId, factoryKey, line.id) : boardHref;

  if (!canAct("factories", "update") || !lineId || (factory && !line)) {
    return <Navigate to={boardHref} replace />;
  }

  return (
    <div className="h-full min-h-0" data-testid="risk-score-setup-page">
      <RiskScoreSetupDialog
        organizationId={organizationId}
        factoryId={factoryId}
        githubIntegrationId={factory?.onboarding?.vcsIntegrationId?.trim() ?? ""}
        appRepository={factory?.onboarding?.appRepository?.trim() ?? ""}
        backlogRepository={factory?.onboarding?.backlogRepository?.trim() ?? ""}
        defaultBranch={factory?.onboarding?.defaultBranch?.trim() ?? ""}
        onClose={() => navigate(returnHref)}
        onCreated={() => navigate(returnHref)}
      />
    </div>
  );
}
