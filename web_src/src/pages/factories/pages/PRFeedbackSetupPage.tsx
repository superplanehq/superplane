import { usePermissions } from "@/contexts/usePermissions";
import { usePageTitle } from "@/hooks/usePageTitle";
import { Navigate, useNavigate, useParams } from "react-router";

import { useFactoriesLayout, type ResolvedFactoriesLayout } from "../layout/factoriesLayoutContext";
import {
  factoryHomePath,
  factoryLineDetailPath,
  firstFactoryLineId,
  type PRFeedbackSetupKind,
} from "../lib/factoryPagePaths";
import { ChecksPRFeedbackSetupDialog } from "./ChecksPRFeedbackSetupDialog";
import { ConflictsPRFeedbackSetupDialog } from "./ConflictsPRFeedbackSetupDialog";
import { DiscussionPRFeedbackSetupDialog } from "./DiscussionPRFeedbackSetupDialog";
import { PR_FEEDBACK_SETTINGS_COPY, prFeedbackSourceById, type PRFeedbackSource } from "./prFeedbackSettingsModel";

export function DiscussionPRFeedbackSetupPage() {
  return <PRFeedbackSetupPage kind="comments" />;
}

export function ChecksPRFeedbackSetupPage() {
  return <PRFeedbackSetupPage kind="checks" />;
}

export function ConflictsPRFeedbackSetupPage() {
  return <PRFeedbackSetupPage kind="conflicts" />;
}

function PRFeedbackSetupPage({ kind }: { kind: PRFeedbackSetupKind }) {
  const layout = useFactoriesLayout();
  const { canAct } = usePermissions();
  const { lineId } = useParams<{ lineId?: string }>();
  const navigate = useNavigate();
  const model = resolvePRFeedbackSetupModel(kind, layout, canAct("factories", "update"), lineId);

  usePageTitle(model.titleParts);

  if (!model.dialog) {
    return <Navigate to={model.redirectTo} replace />;
  }

  return (
    <PRFeedbackSetupDialogs
      {...model.dialog}
      onClose={() => navigate(model.returnHref)}
      onCreated={() => navigate(model.returnHref)}
    />
  );
}

function resolvePRFeedbackSetupModel(
  kind: PRFeedbackSetupKind,
  layout: ResolvedFactoriesLayout,
  canUpdate: boolean,
  lineId: string | undefined,
): {
  titleParts: string[];
  redirectTo: string;
  returnHref: string;
  dialog?: {
    kind: PRFeedbackSetupKind;
    organizationId: string;
    factoryId: string;
    githubIntegrationId: string;
    repository: string;
    source: PRFeedbackSource;
  };
} {
  const { organizationId, factoryId, routeSegment, factory } = layout;
  const bindings = factoryPRFeedbackSetupBindings(factory);
  const boardHref = factoryHomePath(organizationId, routeSegment, firstFactoryLineId(factory));
  const line = bindings.lines.find((entry) => entry.id === lineId);
  const returnHref = line?.id ? factoryLineDetailPath(organizationId, routeSegment, line.id) : boardHref;
  const source = prFeedbackSourceById(prFeedbackSetupSourceId(kind));
  const titleParts = [prFeedbackSetupPageTitle(kind), bindings.workspaceName];
  const redirectTo = prFeedbackSetupRedirect({
    kind,
    canUpdate,
    lineId,
    factoryPresent: Boolean(factory),
    linePresent: Boolean(line),
    sourcePresent: Boolean(source),
    boardHref,
    returnHref,
  });

  if (redirectTo || !source) {
    return { titleParts, redirectTo: redirectTo ?? returnHref, returnHref };
  }

  return {
    titleParts,
    redirectTo: "",
    returnHref,
    dialog: {
      kind,
      organizationId,
      factoryId,
      githubIntegrationId: bindings.githubIntegrationId,
      repository: bindings.repository,
      source,
    },
  };
}

function factoryPRFeedbackSetupBindings(factory: ResolvedFactoriesLayout["factory"]) {
  return {
    workspaceName: factory?.name ?? "Workspace",
    lines: factory?.lines ?? [],
    githubIntegrationId: factory?.onboarding?.vcsIntegrationId?.trim() ?? "",
    repository: factory?.onboarding?.appRepository?.trim() ?? "",
  };
}

function prFeedbackSetupSourceId(kind: PRFeedbackSetupKind) {
  if (kind === "checks") {
    return "checks";
  }
  if (kind === "conflicts") {
    return "conflicts";
  }
  return "discussion";
}

function prFeedbackSetupPageTitle(kind: PRFeedbackSetupKind): string {
  if (kind === "checks") {
    return PR_FEEDBACK_SETTINGS_COPY.wizardPageTitleChecks;
  }
  if (kind === "conflicts") {
    return PR_FEEDBACK_SETTINGS_COPY.wizardPageTitleConflicts;
  }
  return PR_FEEDBACK_SETTINGS_COPY.wizardPageTitleComments;
}

function prFeedbackSetupRedirect(input: {
  kind: PRFeedbackSetupKind;
  canUpdate: boolean;
  lineId?: string;
  factoryPresent: boolean;
  linePresent: boolean;
  sourcePresent: boolean;
  boardHref: string;
  returnHref: string;
}): string | undefined {
  if (!input.canUpdate || !input.lineId || (input.factoryPresent && !input.linePresent)) {
    return input.boardHref;
  }
  if (!input.sourcePresent) {
    return input.returnHref;
  }
  return undefined;
}

function PRFeedbackSetupDialogs({
  kind,
  organizationId,
  factoryId,
  githubIntegrationId,
  repository,
  source,
  onClose,
  onCreated,
}: {
  kind: PRFeedbackSetupKind;
  organizationId: string;
  factoryId: string;
  githubIntegrationId: string;
  repository: string;
  source: PRFeedbackSource;
  onClose: () => void;
  onCreated: () => void;
}) {
  const shared = {
    organizationId,
    factoryId,
    githubIntegrationId,
    repository,
    source,
    onClose,
    onCreated,
  };

  return (
    <div className="h-full min-h-0" data-testid={`pr-feedback-setup-page-${kind}`}>
      {kind === "checks" ? (
        <ChecksPRFeedbackSetupDialog {...shared} />
      ) : kind === "conflicts" ? (
        <ConflictsPRFeedbackSetupDialog
          organizationId={organizationId}
          factoryId={factoryId}
          repository={repository}
          source={source}
          onClose={onClose}
          onCreated={onCreated}
        />
      ) : (
        <DiscussionPRFeedbackSetupDialog {...shared} />
      )}
    </div>
  );
}
