import { Link } from "@/components/Link/link";
import { Button } from "@/components/ui/button";
import { ArrowLeft } from "lucide-react";
import { FactoryAppCanvasTitleEditor } from "./FactoryAppCanvasTitleEditor";
import { FactoryAppCanvasViewActions } from "./FactoryAppCanvasViewActions";
import { FactoryAppCanvasMoreOptions } from "./FactoryAppCanvasMoreOptions";
import {
  FactoryAppCanvasWorkspaceToggles,
  type FactoryAppCanvasWorkspaceTogglesProps,
} from "./FactoryAppCanvasWorkspaceToggles";

type FactoryAppCanvasWorkspaceChrome = FactoryAppCanvasWorkspaceTogglesProps & {
  onViewYaml: () => void;
  onEditWithLocalAgent: () => void;
};

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
  onOpenVisualEditor?: () => void;
  workspace?: FactoryAppCanvasWorkspaceChrome;
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
  onOpenVisualEditor,
  workspace,
}: FactoryAppCanvasHeaderProps) {
  const renameEnabled = Boolean(isConfigure && canRename && onDraftTitleChange);

  return (
    <div
      className="flex shrink-0 items-start justify-between gap-4 border-b border-border px-5 py-3"
      data-testid="factory-app-canvas-header"
      data-editing={isConfigure ? "true" : undefined}
    >
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
          {isConfigure ? (
            <span className="text-[13px] font-medium text-foreground" data-testid="factory-app-editing-badge">
              Editing
            </span>
          ) : null}
        </div>
        <p className="mt-0.5 text-[13px] text-muted-foreground">{subtitle}</p>
      </div>
      {isConfigure ? (
        <FactoryAppCanvasConfigureActions
          configureBusy={configureBusy}
          workspace={workspace}
          onDiscard={onDiscard}
          onSave={onSave}
        />
      ) : onOpenVisualEditor ? (
        <FactoryAppCanvasViewActions onOpenVisualEditor={onOpenVisualEditor} />
      ) : null}
    </div>
  );
}

function FactoryAppCanvasConfigureActions({
  configureBusy,
  workspace,
  onDiscard,
  onSave,
}: {
  configureBusy: boolean;
  workspace?: FactoryAppCanvasWorkspaceChrome;
  onDiscard: () => void;
  onSave: () => void;
}) {
  return (
    <div className="flex shrink-0 flex-col items-end gap-2 pt-0.5">
      <div className="flex items-center gap-2">
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
      {workspace ? (
        <div className="flex items-center gap-1.5">
          <FactoryAppCanvasWorkspaceToggles
            agentOpen={workspace.agentOpen}
            componentsOpen={workspace.componentsOpen}
            onAgentOpenChange={workspace.onAgentOpenChange}
            onComponentsOpenChange={workspace.onComponentsOpenChange}
          />
          <FactoryAppCanvasMoreOptions
            onViewYaml={workspace.onViewYaml}
            onEditWithLocalAgent={workspace.onEditWithLocalAgent}
          />
        </div>
      ) : null}
    </div>
  );
}
