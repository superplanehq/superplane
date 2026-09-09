import { Link } from "@/components/Link/link";
import { usePermissions } from "@/contexts/usePermissions";
import { usePageTitle } from "@/hooks/usePageTitle";
import { ArrowLeft } from "lucide-react";
import { Navigate, useNavigate, useParams } from "react-router";

import { useFactoriesLayout } from "../layout/factoriesLayoutContext";
import { factoryHomePath, factoryLineDetailPath, firstFactoryLineId } from "../lib/factoryPagePaths";
import { ChecksPRFeedbackSetupDialog } from "./ChecksPRFeedbackSetupDialog";
import { DiscussionPRFeedbackSetupDialog } from "./DiscussionPRFeedbackSetupDialog";
import { factoryContentBodyClassName } from "./factoryPageLayoutStyles";
import { prFeedbackSourceById } from "./prFeedbackSettingsModel";

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
  const pageTitle = kind === "checks" ? "Fix pull request checks" : "Address PR feedback";

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
      <Link
        href={returnHref}
        className="mb-6 inline-flex items-center gap-1.5 text-[13px] text-muted-foreground hover:text-foreground"
      >
        <ArrowLeft className="h-4 w-4" aria-hidden />
        Board
      </Link>

      <div className="mx-auto w-full max-w-xl">
        {kind === "checks" ? (
          <ChecksPRFeedbackSetupDialog
            open
            organizationId={organizationId}
            factoryId={factoryId}
            repository={repository}
            source={source}
            onClose={close}
            onCreated={onCreated}
            presentation="page"
          />
        ) : (
          <DiscussionPRFeedbackSetupDialog
            open
            organizationId={organizationId}
            factoryId={factoryId}
            repository={repository}
            source={source}
            onClose={close}
            onCreated={onCreated}
            presentation="page"
          />
        )}
      </div>
    </div>
  );
}
