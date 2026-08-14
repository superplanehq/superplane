import type { FactoriesFactoryLine, FactoryApp, FactoryLineStep } from "@/api-client";
import { Link } from "@/components/Link/link";
import { usePermissions } from "@/contexts/usePermissions";
import { useFactoryApps } from "@/hooks/useFactoryData";
import { usePageTitle } from "@/hooks/usePageTitle";
import { ArrowLeft } from "lucide-react";
import { useMemo } from "react";
import { Navigate, useParams } from "react-router";
import { FactoryLineForm } from "../FactoryLineForm";
import { useFactoriesLayout } from "../layout/factoriesLayoutContext";
import { factoryLineDetailPath, linesPath } from "../lib/factoryPagePaths";
import { useFactoryLineEditActions } from "../useFactoryLineEditActions";
import {
  factoryCardClassName,
  factoryContentBodyClassName,
  factoryPageTitleClassName,
} from "./factoryPageLayoutStyles";
import { cn } from "@/lib/utils";

export function FactoryLineEditPage() {
  const { organizationId, factoryId, factoryKey, factory } = useFactoriesLayout();
  const { canAct } = usePermissions();
  const { lineId } = useParams<{ lineId?: string }>();

  const isCreate = !lineId;
  const canUpdate = canAct("factories", "update");

  const { data: factoryApps = [], isLoading: appsLoading } = useFactoryApps(organizationId, factoryId);

  const line = useMemo(() => {
    if (isCreate || !lineId) {
      return null;
    }
    return factory?.lines?.find((entry) => entry.id === lineId) ?? null;
  }, [factory?.lines, isCreate, lineId]);

  const linesHref = linesPath(organizationId, factoryKey);
  const returnHref = line?.id ? factoryLineDetailPath(organizationId, factoryKey, line.id) : linesHref;
  const actions = useFactoryLineEditActions(organizationId, factoryId, returnHref, isCreate, lineId);

  const pageTitleBase = resolvePageTitleBase(isCreate, line?.name);
  usePageTitle([pageTitleBase, factory?.name ?? "Workspace"]);

  if (!canUpdate) {
    return <Navigate to={linesHref} replace />;
  }

  // FactoriesLayout guarantees `factory` is loaded before rendering us, so a
  // missing line means the URL points at a line that doesn't exist — redirect
  // immediately instead of flashing an empty "Edit line" form.
  if (!isCreate && factory && !line) {
    return <Navigate to={linesHref} replace />;
  }

  // Show a loading state while apps are still fetching so the form isn't
  // mounted with an empty apps list. `FactoriesLayout` already guarantees
  // `factory` is loaded before rendering us, so apps are the only remaining
  // dependency.
  const isInitialLoading = appsLoading;

  return (
    <div className={factoryContentBodyClassName} data-testid="factory-line-edit-page">
      <Link
        href={returnHref}
        className="mb-6 inline-flex items-center gap-1.5 text-[13px] text-muted-foreground hover:text-foreground"
      >
        <ArrowLeft className="h-4 w-4" aria-hidden />
        Lines
      </Link>

      {isInitialLoading ? (
        <p className="text-[13px] text-muted-foreground">Loading…</p>
      ) : (
        <LineEditCard
          isCreate={isCreate}
          line={line}
          organizationId={organizationId}
          factoryApps={factoryApps}
          isSaving={actions.isSaving}
          onSave={actions.handleSave}
          onCancel={actions.navigateToFactory}
        />
      )}
    </div>
  );
}

function resolvePageTitleBase(isCreate: boolean, lineName: string | undefined) {
  if (isCreate) return "New Line";
  return lineName ?? "Edit Line";
}

interface LineEditCardProps {
  isCreate: boolean;
  line: FactoriesFactoryLine | null;
  organizationId: string;
  factoryApps: FactoryApp[];
  isSaving: boolean;
  onSave: (input: { name: string; steps: FactoryLineStep[] }) => Promise<void>;
  onCancel: () => void;
}

function LineEditCard({ isCreate, line, organizationId, factoryApps, isSaving, onSave, onCancel }: LineEditCardProps) {
  const title = isCreate ? "Configure the line" : (line?.name ?? "Edit line");

  return (
    <div className={cn("px-6 py-8 sm:px-8", factoryCardClassName)}>
      <h1 className={factoryPageTitleClassName}>{title}</h1>
      {isCreate ? (
        <p className="mt-2 text-[13px] text-muted-foreground">
          Define the apps and triggers that run when work is dispatched to this line.
        </p>
      ) : null}

      <div className="mt-8 border-t border-border pt-8">
        <FactoryLineForm
          organizationId={organizationId}
          apps={factoryApps}
          initialName={line?.name}
          initialSteps={line?.steps}
          isSaving={isSaving}
          submitLabel={isCreate ? "Create" : "Save"}
          errorMessage={isCreate ? "Failed to create line" : "Failed to update line"}
          onCancel={onCancel}
          onSave={onSave}
        />
      </div>
    </div>
  );
}
