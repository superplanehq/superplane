import { usePermissions } from "@/contexts/usePermissions";
import { usePageTitle } from "@/hooks/usePageTitle";
import { Navigate, useNavigate, useParams } from "react-router";

import { useFactoriesLayout } from "../layout/factoriesLayoutContext";
import { factoryHomePath, factoryLineDetailPath, firstFactoryLineId } from "../lib/factoryPagePaths";
import { ChecksPRFeedbackSetupDialog } from "./ChecksPRFeedbackSetupDialog";
import { DiscussionPRFeedbackSetupDialog } from "./DiscussionPRFeedbackSetupDialog";
import { factoryContentBodyClassName } from "./factoryPageLayoutStyles";
import { PR_FEEDBACK_SETTINGS_COPY, prFeedbackSourceById } from "./prFeedbackSettingsModel";

export function DiscussionPRFeedbackSetupPage() {
  return <PRFeedbackSetupPage kind="comments" />;
}

export function ChecksPRFeedbackSetupPage() {
  return <PRFeedbackSetupPage kind="checks" />;
}

function PRFeedbackSetupPage({ kind }: { kind: "comments" | "checks" }) {
  const { organizationId, factoryId, factoryKey, factory } = useFactoriesLayout();
  const { canAct } = usePermissions();
  const { lineId } = useParams<{ lineId?: string }>();
  const navigate = useNavigate();
  const canUpdate = canAct("factories", "update");
  const boardHref = factoryHomePath(organizationId, factoryKey, firstFactoryLineId(factory));
  const line = factory?.lines?.find((entry) => entry.id === lineId);
  const returnHref = line?.id ? factoryLineDetailPath(organizationId, factoryKey, line.id) : boardHref;
  const repository = factory?.onboarding?.appRepository?.trim() ?? "";
  const source = prFeedbackSourceById(kind === "checks" ? "checks" : "discussion");
  const pageTitle =
    kind === "checks"
      ? PR_FEEDBACK_SETTINGS_COPY.wizardPageTitleChecks
      : PR_FEEDBACK_SETTINGS_COPY.wizardPageTitleComments;

  usePageTitle([pageTitle, factory?.name ?? "Workspace"]);

  if (!canUpdate) {
    return <Navigate to={boardHref} replace />;
  }

  if (!lineId || (factory && !line)) {
    return <Navigate to={boardHref} replace />;
  }

  if (!source) {
    return <Navigate to={returnHref} replace />;
  }

  const close = () => navigate(returnHref);
  const onCreated = () => navigate(returnHref);

  return (
    <div className={factoryContentBodyClassName} data-testid={`pr-feedback-setup-page-${kind}`}>
      <div className="mx-auto w-full max-w-2xl">
        {kind === "checks" ? (
          <ChecksPRFeedbackSetupDialog
            organizationId={organizationId}
            factoryId={factoryId}
            repository={repository}
            source={source}
            onClose={close}
            onCreated={onCreated}
          />
        ) : (
          <DiscussionPRFeedbackSetupDialog
            organizationId={organizationId}
            factoryId={factoryId}
            repository={repository}
            source={source}
            onClose={close}
            onCreated={onCreated}
          />
        )}
      </div>
    </div>
  );
}
