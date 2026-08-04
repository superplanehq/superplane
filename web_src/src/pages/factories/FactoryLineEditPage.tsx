import { Link } from "@/components/Link/link";
import { Text } from "@/components/Text/text";
import { ArrowLeft } from "lucide-react";
import { useParams } from "react-router-dom";
import { FactoryLineForm } from "./FactoryLineForm";
import { factoryFormCardClassName, factoryPageContentClassName } from "./lib/factoryPageStyles";
import { FactoryPageShell } from "./FactoryPageShell";
import { useFactoryLineEditPage } from "./useFactoryLineEditPage";

export function FactoryLineEditPage() {
  const { organizationId, factoryId, lineId } = useParams<{
    organizationId: string;
    factoryId: string;
    lineId: string;
  }>();

  if (!organizationId || !factoryId) {
    return null;
  }

  return <FactoryLineEditPageContent organizationId={organizationId} factoryId={factoryId} lineId={lineId} />;
}

function FactoryLineEditPageContent({
  organizationId,
  factoryId,
  lineId,
}: {
  organizationId: string;
  factoryId: string;
  lineId?: string;
}) {
  const page = useFactoryLineEditPage(organizationId, factoryId, lineId);

  if (page.kind === "redirect") {
    return page.element;
  }

  return (
    <FactoryPageShell backHref={page.factoryHref} backLabel={page.factory?.name ?? "Factory"}>
      {page.isLoading ? (
        <div className="px-8 py-6">
          <Text className="text-sm text-gray-500">Loading…</Text>
        </div>
      ) : page.factory ? (
        <div className={factoryPageContentClassName}>
          <Link
            href={page.factoryHref}
            className="mb-6 inline-flex items-center gap-1.5 text-sm text-gray-500 hover:text-slate-900 dark:text-gray-400 dark:hover:text-gray-100"
          >
            <ArrowLeft className="h-4 w-4" aria-hidden />
            {page.factory.name}
          </Link>

          <div className={factoryFormCardClassName}>
            <h1 className="text-2xl font-semibold tracking-tight text-slate-900 dark:text-gray-100">
              {page.isCreate ? "Configure the line" : page.line?.name}
            </h1>
            {page.isCreate ? (
              <p className="mt-2 text-sm text-gray-500 dark:text-gray-400">
                Define the apps and triggers that run when work is dispatched to this line.
              </p>
            ) : null}

            <div className="mt-8 border-t border-slate-200 pt-8 dark:border-gray-700/70">
              <FactoryLineForm
                organizationId={page.organizationId}
                apps={page.factoryApps}
                initialName={page.line?.name}
                initialSteps={page.line?.steps}
                isSaving={page.isSaving}
                submitLabel={page.isCreate ? "Create" : "Save"}
                errorMessage={page.isCreate ? "Failed to create line" : "Failed to update line"}
                onCancel={page.navigateToFactory}
                onSave={page.handleSave}
              />
            </div>
          </div>
        </div>
      ) : null}
    </FactoryPageShell>
  );
}
