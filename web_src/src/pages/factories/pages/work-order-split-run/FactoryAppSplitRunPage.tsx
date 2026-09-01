import { cn } from "@/lib/utils";

import { FactoryAppCanvasHeader } from "../FactoryAppCanvasHeader";
import { CompactLineCanvas } from "./CompactLineCanvas";
import { PhaseLogCard } from "./PhaseLogCard";
import { SplitRunLogHeader } from "./SplitRunLogHeader";
import { runningSplitRunPhaseId } from "./followLogScroll";
import { splitRunMissingCopy } from "./splitRunPageModel";
import { useFactoryAppSplitRunPage } from "./useFactoryAppSplitRunPage";
import { useFollowLogScroll } from "./useFollowLogScroll";

/**
 * Copy of the factory Automation Run page. The body is a resizable split:
 * the selected canvas log on the left, that canvas on the right.
 */
export function FactoryAppSplitRunPage() {
  const model = useFactoryAppSplitRunPage();
  if (!model.fixture || model.liveError) {
    return (
      <SplitRunMissingPage
        back={model.back}
        editHref={model.editHref}
        failed={model.liveError}
        isLoading={model.isLoading}
        subtitle={model.subtitle}
      />
    );
  }
  return <SplitRunLoadedPage model={model} />;
}

function SplitRunMissingPage({
  back,
  editHref,
  failed,
  isLoading,
  subtitle,
}: {
  back: { href: string; label: string };
  editHref: string;
  failed: boolean;
  isLoading: boolean;
  subtitle: string;
}) {
  const copy = splitRunMissingCopy(isLoading, failed);
  return (
    <div
      className="absolute inset-0 flex flex-col bg-background"
      data-testid="factory-app-split-run-page"
      data-state={isLoading && !failed ? "loading" : failed ? "failed" : "not-found"}
    >
      <FactoryAppCanvasHeader
        backHref={back.href}
        backLabel={back.label}
        title={copy.title}
        subtitle={subtitle}
        isConfigure={false}
        configureBusy={false}
        onDiscard={() => undefined}
        onSave={() => undefined}
        editHref={editHref}
      />
      <p className="px-6 py-8 text-sm text-muted-foreground">{copy.body}</p>
    </div>
  );
}

function SplitRunLoadedPage({ model }: { model: ReturnType<typeof useFactoryAppSplitRunPage> }) {
  const follow = useFollowLogScroll(runningSplitRunPhaseId(model.fixture?.phases ?? []), model.stream?.length ?? 0);
  return (
    <div className="absolute inset-0 flex flex-col bg-background" data-testid="factory-app-split-run-page">
      <FactoryAppCanvasHeader
        backHref={model.back.href}
        backLabel={model.back.label}
        title={model.canvas.title}
        subtitle={model.subtitle}
        isConfigure={false}
        configureBusy={false}
        onDiscard={() => undefined}
        onSave={() => undefined}
        editHref={model.editHref}
      />
      <div ref={model.split.containerRef} className="flex min-h-0 flex-1 overflow-hidden">
        <aside
          className="flex min-h-0 min-w-[12rem] flex-col overflow-hidden border-r border-border bg-muted/25"
          style={{ width: `${model.split.percent}%` }}
          aria-label="Automations"
        >
          <SplitRunLogHeader
            following={follow.following}
            onFollowingChange={follow.setFollowing}
            className="px-4 pt-3 pb-2"
          />
          <ol
            ref={follow.scrollRef}
            onScroll={follow.onScroll}
            className="min-h-0 min-w-0 flex-1 list-none overflow-x-hidden overflow-y-auto px-4 pb-3"
            data-testid="split-run-log-scroll"
          >
            <li className="min-w-0">
              <PhaseLogCard
                phase={model.phase}
                expanded
                collapsible={false}
                stream={model.stream}
                selectedNodeId={model.nodeId}
                onSelectNode={model.setNodeId}
                organizationId={model.organizationId}
                canvasId={model.phase.appId}
                editHref={model.editHref}
              />
            </li>
          </ol>
        </aside>
        <div
          role="separator"
          aria-orientation="vertical"
          aria-label="Resize log and canvas"
          data-testid="split-run-resize-handle"
          onPointerDown={model.split.startResize}
          className="group relative z-10 w-2 shrink-0 cursor-col-resize bg-transparent"
        >
          <div
            aria-hidden
            className={cn(
              "pointer-events-none absolute inset-y-0 left-1/2 w-px -translate-x-1/2 bg-transparent transition-colors group-hover:bg-border",
              model.split.isResizing && "bg-border",
            )}
          />
        </div>
        <section className="flex min-h-0 min-w-[12rem] flex-1 flex-col" aria-label="Run">
          <CompactLineCanvas
            canvas={model.canvas}
            selectedId={model.nodeId}
            onSelect={model.setNodeId}
            showHeader={false}
            nodeEditHref={model.nodeEditHref}
          />
        </section>
      </div>
    </div>
  );
}
