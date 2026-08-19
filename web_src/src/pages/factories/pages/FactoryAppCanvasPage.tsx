import { AppPage } from "@/pages/app";
import { usePageTitle } from "@/hooks/usePageTitle";
import { useFactoriesLayout } from "../layout/factoriesLayoutContext";
import { FactoryAppCanvasHeader } from "./FactoryAppCanvasHeader";
import { FactoryAppCanvasRedirect } from "./factoryAppCanvasGuards";
import { useFactoryAppCanvasPageModel } from "./useFactoryAppCanvasPageModel";

/**
 * Factory-shell embed for a factory-owned app/canvas. Keeps the workspace
 * sidebar and a route-aware back header. View mode is read-only; `?configure=1`
 * opens Configure (edit mode) with Discard / Save.
 */
export function FactoryAppCanvasPage() {
  const { factory } = useFactoriesLayout();
  const model = useFactoryAppCanvasPageModel();

  // `model.title` is already computed for the visible header (falls back to
  // "Untitled automation" while the canvas name loads); reuse it here so the
  // tab title and on-page heading always agree.
  usePageTitle([model.title, factory?.name ?? "Workspace"]);

  if (model.shouldRedirect) {
    return <FactoryAppCanvasRedirect organizationId={model.organizationId} factoryKey={model.factoryKey} />;
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
      />
      <div className="relative min-h-0 flex-1 overflow-hidden">
        {model.canvasLoading && !model.canvas ? (
          <p className="p-5 text-[13px] text-muted-foreground">Loading…</p>
        ) : (
          <AppPage
            factoryEmbed
            factoryConfigure={model.isConfigure}
            factoryConfigureActionsRef={model.configureActionsRef}
            onFactoryConfigureBusyChange={model.handleConfigureBusyChange}
            onFactoryConfigureDone={model.handleConfigureDone}
          />
        )}
      </div>
    </div>
  );
}
