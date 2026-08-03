import { Link } from "@/components/Link/link";
import { Text } from "@/components/Text/text";
import { usePermissions } from "@/contexts/usePermissions";
import {
  useCreateFactoryLine,
  useFactory,
  useFactoryApps,
  useUpdateFactoryLine,
  type FactoryLineStep,
} from "@/hooks/useFactoryData";
import { usePageTitle } from "@/hooks/usePageTitle";
import { useReportPageReady } from "@/hooks/useReportPageReady";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { ArrowLeft } from "lucide-react";
import { useMemo } from "react";
import { Navigate, useNavigate, useParams } from "react-router-dom";
import { FactoryLineForm } from "./FactoryLineForm";
import { factoryFormCardClassName, factoryPageContentClassName } from "./factoryPageStyles";
import { FactoryPageShell } from "./FactoryPageShell";

export function FactoryLineEditPage() {
  const navigate = useNavigate();
  const { organizationId, factoryId, lineId } = useParams<{
    organizationId: string;
    factoryId: string;
    lineId: string;
  }>();
  const { canAct } = usePermissions();
  const isCreate = !lineId;

  const {
    data: factory,
    isLoading: factoryLoading,
    error: factoryError,
  } = useFactory(organizationId ?? "", factoryId ?? "");
  const { data: factoryApps = [], isLoading: appsLoading } = useFactoryApps(organizationId ?? "", factoryId ?? "");

  const createFactoryLine = useCreateFactoryLine(organizationId ?? "", factoryId ?? "");
  const updateFactoryLine = useUpdateFactoryLine(organizationId ?? "", factoryId ?? "");

  const line = useMemo(() => {
    if (isCreate || !lineId) {
      return null;
    }
    return factory?.lines?.find((entry) => entry.id === lineId) ?? null;
  }, [factory?.lines, isCreate, lineId]);

  usePageTitle([isCreate ? "New Line" : (line?.name ?? "Edit Line"), factory?.name ?? "Factory"]);

  const isLoading = factoryLoading || appsLoading;
  const canUpdate = canAct("factories", "update");

  useReportPageReady(!factoryLoading && Boolean(factory) && (isCreate || Boolean(line)), {
    failed: Boolean(factoryError) || (!isCreate && !factoryLoading && !line),
  });

  if (!organizationId || !factoryId) {
    return null;
  }

  if (!canUpdate) {
    return <Navigate to={`/${organizationId}/factories/${factoryId}`} replace />;
  }

  if (!factoryLoading && factoryError) {
    return <Navigate to={`/${organizationId}/factories`} replace />;
  }

  if (!isCreate && !factoryLoading && factory && !line) {
    return <Navigate to={`/${organizationId}/factories/${factoryId}`} replace />;
  }

  const factoryHref = `/${organizationId}/factories/${factoryId}`;

  const handleSave = async (input: { name: string; steps: FactoryLineStep[] }) => {
    try {
      if (isCreate) {
        await createFactoryLine.mutateAsync(input);
        showSuccessToast("Line created.");
      } else if (lineId) {
        await updateFactoryLine.mutateAsync({
          lineId,
          name: input.name,
          steps: input.steps,
        });
        showSuccessToast("Line updated.");
      }
      navigate(factoryHref);
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, isCreate ? "Failed to create line" : "Failed to update line"));
      throw error;
    }
  };

  return (
    <FactoryPageShell backHref={factoryHref} backLabel={factory?.name ?? "Factory"}>
      {isLoading ? (
        <div className="px-8 py-6">
          <Text className="text-sm text-gray-500">Loading…</Text>
        </div>
      ) : factory ? (
        <div className={factoryPageContentClassName}>
          <Link
            href={factoryHref}
            className="mb-6 inline-flex items-center gap-1.5 text-sm text-gray-500 hover:text-slate-900 dark:text-gray-400 dark:hover:text-gray-100"
          >
            <ArrowLeft className="h-4 w-4" aria-hidden />
            {factory.name}
          </Link>

          <div className={factoryFormCardClassName}>
            <h1 className="text-2xl font-semibold tracking-tight text-slate-900 dark:text-gray-100">
              {isCreate ? "Configure the line" : line?.name}
            </h1>
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
                isSaving={createFactoryLine.isPending || updateFactoryLine.isPending}
                submitLabel={isCreate ? "Create" : "Save"}
                errorMessage={isCreate ? "Failed to create line" : "Failed to update line"}
                onCancel={() => navigate(factoryHref)}
                onSave={handleSave}
              />
            </div>
          </div>
        </div>
      ) : null}
    </FactoryPageShell>
  );
}
