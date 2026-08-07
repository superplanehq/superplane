import type { FactoriesFactoryLine, FactoryApp, FactoryLineStep } from "@/api-client";
import { Link } from "@/components/Link/link";
import { Text } from "@/components/Text/text";
import { usePermissions } from "@/contexts/usePermissions";
import { useFactoryApps } from "@/hooks/useFactoryData";
import { usePageTitle } from "@/hooks/usePageTitle";
import { ArrowLeft } from "lucide-react";
import { useMemo } from "react";
import { Navigate, useParams } from "react-router-dom";
import { FactoryLineForm } from "../FactoryLineForm";
import { useFactoriesLayout } from "../layout/factoriesLayoutContext";
import { automationsPath } from "../lib/factoryPagePaths";
import { useFactoryLineEditActions } from "../useFactoryLineEditActions";
import { factoryContentBodyClassName } from "./factoryPageLayoutStyles";

export function FactoryLineEditPage() {
  const { organizationId, factoryId, factory } = useFactoriesLayout();
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

  const automationsHref = automationsPath(organizationId, factoryId);
  const returnHref = line?.id ? `${automationsHref}/${line.id}` : automationsHref;
  const actions = useFactoryLineEditActions(organizationId, factoryId, returnHref, isCreate, lineId);

  const pageTitleBase = resolvePageTitleBase(isCreate, line?.name);
  usePageTitle([pageTitleBase, factory?.name ?? "Workspace"]);

  if (!canUpdate) {
    return <Navigate to={automationsHref} replace />;
  }

  // FactoriesLayout guarantees `factory` is loaded before rendering us, so a
  // missing line means the URL points at a line that doesn't exist — redirect
  // immediately instead of flashing an empty "Edit line" form.
  if (!isCreate && factory && !line) {
    return <Navigate to={automationsHref} replace />;
  }

  const isInitialLoading = appsLoading && !factory;

  return (
    <div className={factoryContentBodyClassName} data-testid="factory-line-edit-page">
      <Link
        href={returnHref}
        className="mb-6 inline-flex items-center gap-1.5 text-sm text-gray-500 hover:text-slate-900 dark:text-gray-400 dark:hover:text-gray-100"
      >
        <ArrowLeft className="h-4 w-4" aria-hidden />
        Automations
      </Link>

      {isInitialLoading ? (
        <Text className="text-sm text-gray-500">Loading…</Text>
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
    <div className="rounded-lg border border-slate-950/10 bg-white px-6 py-8 sm:px-8 dark:border-gray-700/70 dark:bg-gray-900">
      <h1 className="text-2xl font-semibold tracking-tight text-slate-900 dark:text-gray-100">{title}</h1>
      {isCreate ? (
        <p className="mt-2 text-sm text-gray-500 dark:text-gray-400">
          Define the apps and triggers that run when work is dispatched to this line.
        </p>
      ) : null}

      <div className="mt-8 border-t border-slate-200 pt-8 dark:border-gray-700/70">
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
