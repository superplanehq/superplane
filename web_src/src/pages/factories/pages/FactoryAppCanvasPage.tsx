import { AppPage } from "@/pages/app";
import { usePageTitle } from "@/hooks/usePageTitle";
import { Navigate } from "react-router";
import { DEFAULT_SUPERPLANE_BASE_URL, buildAgentEditPrompt } from "../lib/agentEditPrompt";
import { factoryAppSplitRunPath, parseFactoryAppNavFrom } from "../lib/factoryPagePaths";
import { useFactoriesLayout } from "../layout/factoriesLayoutContext";
import { AgentSetupPromptDialog } from "./AgentSetupPromptDialog";
import { FactoryAppCanvasHeader } from "./FactoryAppCanvasHeader";
import { FactoryAppCanvasRedirect } from "./factoryAppCanvasGuards";
import { FactoryCanvasYamlModal } from "./FactoryCanvasYamlModal";
import { useFactoryAppCanvasPageModel } from "./useFactoryAppCanvasPageModel";

/**
 * Factory-shell embed for a factory-owned app/canvas. Configure (`?configure=1`)
 * stays here. Viewing a run (`?run=` without configure) redirects to the
 * split-run page — the line-board popup is the run surface.
 */
export function FactoryAppCanvasPage() {
  const { factory } = useFactoriesLayout();
  const model = useFactoryAppCanvasPageModel();
  const agentPrompt = buildAgentEditPrompt({
    appName: model.title,
    appId: model.appId,
    baseUrl: DEFAULT_SUPERPLANE_BASE_URL,
    runId: model.runId,
    lineId: model.lineId,
  });

  // `model.title` is already computed for the visible header (falls back to
  // "Untitled automation" while the canvas name loads); reuse it here so the
  // tab title and on-page heading always agree.
  usePageTitle([model.title, factory?.name ?? "Workspace"]);

  if (model.shouldRedirect) {
    return <FactoryAppCanvasRedirect organizationId={model.organizationId} factoryKey={model.factoryKey} />;
  }

  if (model.runId && !model.isConfigure) {
    return (
      <Navigate
        to={factoryAppSplitRunPath(model.organizationId, model.factoryKey, model.appId, {
          from: parseFactoryAppNavFrom(model.from),
          lineId: model.lineId ?? undefined,
          runId: model.runId,
          orderNumber: model.orderNumber ?? undefined,
        })}
        replace
      />
    );
  }

  return (
    <div
      className="absolute inset-0 flex flex-col bg-background"
      data-testid="factory-app-canvas-page"
      data-configure={model.isConfigure ? "true" : undefined}
    >
      <FactoryAppCanvasHeader
        backHref={model.back.href}
        backLabel={model.back.label}
        title={model.title}
        subtitle={model.subtitle}
        isConfigure={model.isConfigure}
        configureBusy={model.configureBusy}
        canRename={model.canRename}
        onDraftTitleChange={model.isConfigure ? model.handleDraftTitleChange : undefined}
        onDiscard={model.handleConfigureDiscard}
        onSave={model.handleConfigureSave}
        onOpenVisualEditor={model.storybookEditWorkspace ? model.handleOpenVisualEditor : undefined}
        workspace={
          model.storybookEditWorkspace && model.isConfigure
            ? {
                agentOpen: model.agentOpen,
                componentsOpen: model.componentsOpen,
                onAgentOpenChange: model.handleAgentOpenChange,
                onComponentsOpenChange: model.handleComponentsOpenChange,
                onViewYaml: model.handleViewYaml,
                onEditWithLocalAgent: model.handleEditWithLocalAgent,
              }
            : undefined
        }
      />
      <div className="relative min-h-0 flex-1 overflow-hidden">
        {model.canvasLoading && !model.canvas ? (
          <p className="p-5 text-[13px] text-muted-foreground">Loading…</p>
        ) : (
          <AppPage
            factoryEmbed
            factoryConfigure={model.isConfigure}
            factoryAgentEnabled={model.storybookEditWorkspace && model.isConfigure}
            factoryEditWorkspace={model.storybookEditWorkspace}
            factoryConfigureActionsRef={model.configureActionsRef}
            onFactoryConfigureBusyChange={model.handleConfigureBusyChange}
            onFactoryConfigureDone={model.handleConfigureDone}
            onFactoryConfigureSaved={model.storybookEditWorkspace ? model.handleConfigureSaved : undefined}
          />
        )}
      </div>
      {model.storybookEditWorkspace ? (
        <>
          <FactoryCanvasYamlModal
            open={model.yamlViewOpen}
            onOpenChange={model.handleYamlViewOpenChange}
            canvas={model.canvas}
          />
          <AgentSetupPromptDialog
            open={model.agentPromptOpen}
            onOpenChange={model.handleAgentPromptOpenChange}
            prompt={agentPrompt}
          />
        </>
      ) : null}
    </div>
  );
}
