import { useNavigate, useParams } from "react-router";
import { WorkspacePageHeader } from "../layout/WorkspacePageHeader";
import { parseFactoryAppNavFrom } from "../lib/factoryPagePaths";
import { factoryAutomationRunHistoryPath } from "../lib/factoryAutomationRunHistoryPath";
import { FactoryAutomationRunHistory } from "./automationRunHistory/FactoryAutomationRunHistory";
import { FactoryAppCanvasRedirect } from "./factoryAppCanvasGuards";
import { useFactoryAppCanvasRoute } from "./useFactoryAppCanvasRoute";

/** Run inspection keeps the automation implementation out of the task flow. */
export function FactoryAutomationRunPage() {
  const route = useFactoryAppCanvasRoute();
  const { runId } = useParams<{ runId: string }>();
  const navigate = useNavigate();
  if (route.shouldRedirect) {
    return <FactoryAppCanvasRedirect organizationId={route.organizationId} factoryKey={route.factoryKey} />;
  }
  return (
    <div className="absolute inset-0 flex flex-col bg-background" data-testid="factory-automation-run-page">
      <WorkspacePageHeader
        variant="entity"
        title={route.canvas?.metadata?.name?.trim() || "Run history"}
        backHref={route.back.href}
        backLabel={route.back.label}
      />
      {route.canvasLoading ? (
        <p role="status" className="p-6 text-sm text-muted-foreground">
          Loading run history…
        </p>
      ) : (
        <FactoryAutomationRunHistory
          organizationId={route.organizationId}
          factoryId={route.factoryId}
          canvasId={route.appId}
          selectedRunId={runId ?? null}
          onSelectRun={(id) =>
            navigate(
              factoryAutomationRunHistoryPath(route.organizationId, route.factoryKey, route.appId, id, {
                from: parseFactoryAppNavFrom(route.from),
                lineId: route.lineId ?? undefined,
                orderNumber: route.orderNumber ?? undefined,
              }),
            )
          }
        />
      )}
    </div>
  );
}
