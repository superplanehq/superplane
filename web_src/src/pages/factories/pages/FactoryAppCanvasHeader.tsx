import { Link } from "@/components/Link/link";
import { Button } from "@/components/ui/button";
import { ArrowLeft } from "lucide-react";
import { FactoryAppCanvasTitleEditor } from "./FactoryAppCanvasTitleEditor";
import { FactoryAppCanvasViewActions } from "./FactoryAppCanvasViewActions";

type FactoryAppCanvasHeaderProps = {
  backHref: string;
  backLabel: string;
  title: string;
  subtitle: string;
  isConfigure: boolean;
  configureBusy: boolean;
  canRename?: boolean;
  onDiscard: () => void;
  onSave: () => void;
  /** Local draft only — persisted when the user clicks Save. */
  onDraftTitleChange?: (name: string) => void;
  yamlDisabled?: boolean;
  onOpenVisualEditor?: () => void;
  onAskAgent?: () => void;
  onOpenDesktopAgentSetup?: () => void;
  onViewYaml?: () => void;
};

export function FactoryAppCanvasHeader({
  backHref,
  backLabel,
  title,
  subtitle,
  isConfigure,
  configureBusy,
  canRename = false,
  onDiscard,
  onSave,
  onDraftTitleChange,
  yamlDisabled = false,
  onOpenVisualEditor,
  onAskAgent,
  onOpenDesktopAgentSetup,
  onViewYaml,
}: FactoryAppCanvasHeaderProps) {
  const renameEnabled = Boolean(isConfigure && canRename && onDraftTitleChange);

  return (
    <div className="flex shrink-0 items-start justify-between gap-4 border-b border-border px-5 py-3">
      <div className="min-w-0">
        <Link
          href={backHref}
          className="mb-2 inline-flex items-center gap-1.5 text-[13px] text-muted-foreground transition-colors hover:text-foreground"
          data-testid="factory-app-canvas-back"
        >
          <ArrowLeft className="size-3.5" strokeWidth={1.75} aria-hidden />
          {backLabel}
        </Link>
        <div className="flex flex-wrap items-center gap-2">
          {isConfigure && onDraftTitleChange ? (
            <FactoryAppCanvasTitleEditor
              title={title}
              renameEnabled={renameEnabled}
              configureBusy={configureBusy}
              onDraftTitleChange={onDraftTitleChange}
            />
          ) : (
            <h2
              className="text-[15px] font-semibold tracking-[-0.01em] text-foreground"
              data-testid="factory-app-canvas-title"
            >
              {title}
            </h2>
          )}
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
            onClick={onDiscard}
            data-testid="factory-app-discard"
          >
            Discard changes
          </Button>
          <Button type="button" size="sm" disabled={configureBusy} onClick={onSave} data-testid="factory-app-save">
            Save
          </Button>
        </div>
      ) : onOpenVisualEditor && onAskAgent && onOpenDesktopAgentSetup && onViewYaml ? (
        <FactoryAppCanvasViewActions
          yamlDisabled={yamlDisabled}
          onOpenVisualEditor={onOpenVisualEditor}
          onAskAgent={onAskAgent}
          onOpenDesktopAgentSetup={onOpenDesktopAgentSetup}
          onViewYaml={onViewYaml}
        />
      ) : null}
    </div>
  );
}
