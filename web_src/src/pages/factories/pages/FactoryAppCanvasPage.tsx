import { Link } from "@/components/Link/link";
import { Button } from "@/components/ui/button";
import { useCanvas } from "@/hooks/useCanvasData";
import { useWorkOrder } from "@/hooks/useFactoryData";
import { AppPage, type FactoryConfigureActions } from "@/pages/app";
import { ArrowLeft } from "lucide-react";
import { useCallback, useMemo, useRef, useState } from "react";
import { Navigate, useNavigate, useParams, useSearchParams } from "react-router-dom";
import { useFactoriesLayout } from "../layout/factoriesLayoutContext";
import { resolveFactoryAppBackNav } from "../lib/factoryAppNav";
import { factoryOverviewPath } from "../lib/factoryPagePaths";

/**
 * Factory-shell embed for a factory-owned app/canvas. Keeps the workspace
 * sidebar and a route-aware back header. View mode is read-only; `?configure=1`
 * opens Configure (edit mode) with Discard / Save.
 */
export function FactoryAppCanvasPage() {
  const { organizationId, factoryId, factory } = useFactoriesLayout();
  const { appId = "" } = useParams<{ appId: string }>();
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const configureActionsRef = useRef<FactoryConfigureActions | null>(null);
  const [configureBusy, setConfigureBusy] = useState(false);

  const {
    data: canvas,
    isLoading: canvasLoading,
    error: canvasError,
  } = useCanvas(organizationId, appId, {
    enabled: Boolean(appId),
  });

  const from = searchParams.get("from");
  const lineId = searchParams.get("lineId");
  const orderId = searchParams.get("orderId");
  // Prefer configure=1. Accept legacy edit=1 only until that param is stripped by AppPage.
  const isConfigure = searchParams.get("configure") === "1" || searchParams.get("edit") === "1";

  const lineName = useMemo(() => {
    if (!lineId) return null;
    return factory?.lines?.find((line) => line.id === lineId)?.name ?? null;
  }, [factory?.lines, lineId]);

  const { data: order } = useWorkOrder(organizationId, factoryId, orderId ?? "");

  const back = useMemo(
    () =>
      resolveFactoryAppBackNav(organizationId, factoryId, {
        from,
        appId,
        appName: canvas?.metadata?.name,
        lineId,
        orderId,
        lineName,
        orderTitle: order?.title,
      }),
    [appId, canvas?.metadata?.name, factoryId, from, lineId, lineName, order?.title, organizationId, orderId],
  );

  const handleConfigureDone = useCallback(() => {
    navigate(back.href);
  }, [back.href, navigate]);

  const handleConfigureBusyChange = useCallback((busy: boolean) => {
    setConfigureBusy((prev) => (prev === busy ? prev : busy));
  }, []);

  const canvasFactoryId = canvas?.metadata?.factoryId;
  const belongsToFactory = Boolean(canvasFactoryId && canvasFactoryId === factoryId);

  if (!appId) {
    return <Navigate to={factoryOverviewPath(organizationId, factoryId)} replace />;
  }

  if (!canvasLoading && (canvasError || (canvas && !belongsToFactory))) {
    return <Navigate to={factoryOverviewPath(organizationId, factoryId)} replace />;
  }

  const title = canvas?.metadata?.name?.trim() || "App";
  const description = canvas?.metadata?.description?.trim();
  const subtitle = isConfigure
    ? "Drag steps and reconnect edges to configure this automation."
    : description || `Canvas · ${factory?.name?.trim() || "Workspace"}`;

  return (
    <div
      className="absolute inset-0 flex flex-col bg-background"
      data-testid="factory-app-canvas-page"
      data-configure={isConfigure ? "true" : undefined}
    >
      <div
        className={
          isConfigure
            ? "flex shrink-0 items-start justify-between gap-4 border-b border-border px-5 py-3"
            : "shrink-0 border-b border-border px-5 py-3"
        }
      >
        <div className="min-w-0">
          <Link
            href={back.href}
            className="mb-2 inline-flex items-center gap-1.5 text-[13px] text-muted-foreground transition-colors hover:text-foreground"
            data-testid="factory-app-canvas-back"
          >
            <ArrowLeft className="size-3.5" strokeWidth={1.75} aria-hidden />
            {back.label}
          </Link>
          <div className="flex flex-wrap items-center gap-2">
            <h2 className="text-[15px] font-semibold tracking-[-0.01em] text-foreground">{title}</h2>
            {isConfigure ? (
              <span
                className="rounded-md bg-foreground px-1.5 py-0.5 text-[11px] font-medium text-primary-foreground"
                data-testid="factory-app-edit-mode-badge"
              >
                Edit mode
              </span>
            ) : null}
          </div>
          <p
            className={
              isConfigure ? "mt-1 text-[12px] text-muted-foreground" : "mt-0.5 text-[13px] text-muted-foreground"
            }
          >
            {subtitle}
          </p>
        </div>
        {isConfigure ? (
          <div className="flex shrink-0 items-center gap-2 pt-0.5">
            <Button
              type="button"
              variant="outline"
              size="sm"
              disabled={configureBusy}
              onClick={() => configureActionsRef.current?.discard()}
              data-testid="factory-app-discard"
            >
              Discard changes
            </Button>
            <Button
              type="button"
              size="sm"
              disabled={configureBusy}
              onClick={() => configureActionsRef.current?.save()}
              data-testid="factory-app-save"
            >
              Save
            </Button>
          </div>
        ) : null}
      </div>
      <div className="relative min-h-0 flex-1 overflow-hidden">
        {canvasLoading && !canvas ? (
          <p className="p-5 text-[13px] text-muted-foreground">Loading…</p>
        ) : (
          <AppPage
            factoryEmbed
            factoryConfigure={isConfigure}
            factoryConfigureActionsRef={configureActionsRef}
            onFactoryConfigureBusyChange={handleConfigureBusyChange}
            onFactoryConfigureDone={handleConfigureDone}
          />
        )}
      </div>
    </div>
  );
}
