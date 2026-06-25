import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";
import { FileCode, FilePlus, Plus } from "lucide-react";
import { memo, type ReactNode } from "react";

export type RightSideControlsProps = {
  mode: "live" | "edit";
  /** When true, shows canvas edit controls in the right rail. */
  canvasEditControls?: boolean;
  /** When true, shows console edit controls in the right rail. */
  consoleEditControls?: boolean;
  /** Overlay on the canvas area (default) or embedded beside console content. */
  layout?: "overlay" | "embedded";

  onSidebarOpen?: () => void;
  onAddNote?: () => void | Promise<void>;
  onConsoleAddPanel?: () => void;
  onConsoleOpenYaml?: () => void;
  consoleYamlReadOnly?: boolean;
};

export const RightSideControls = memo(function RightSideControls(props: RightSideControlsProps) {
  if (props.mode === "live") return null;

  const railClassName =
    props.layout === "embedded"
      ? "flex w-9 shrink-0 flex-col items-center gap-1.5 border-l border-slate-950/15 bg-slate-100 py-2"
      : "absolute inset-y-0 right-0 z-10 flex w-9 flex-col items-center gap-1.5 border-l border-slate-950/15 bg-slate-100 py-2";

  return (
    <div className={railClassName}>
      <EditModeButtons {...props} />
    </div>
  );
});

function EditModeButtons({
  canvasEditControls,
  consoleEditControls,
  onSidebarOpen,
  onAddNote,
  onConsoleAddPanel,
  onConsoleOpenYaml,
  consoleYamlReadOnly,
}: RightSideControlsProps) {
  if (consoleEditControls) {
    return (
      <>
        {onConsoleAddPanel ? (
          <ControlButton
            tooltip="Add Panel"
            onClick={onConsoleAddPanel}
            testId="console-add-panel"
            icon={<Plus className="h-3.5 w-3.5" />}
          />
        ) : null}
        {onConsoleOpenYaml ? (
          <ControlButton
            tooltip={
              consoleYamlReadOnly ? "View the console as YAML" : "View, copy, download, or import this console as YAML"
            }
            onClick={onConsoleOpenYaml}
            testId="console-yaml-button"
            ariaLabel={consoleYamlReadOnly ? "View YAML" : "View / Import YAML"}
            icon={<FileCode className="h-3.5 w-3.5" />}
          />
        ) : null}
      </>
    );
  }

  if (canvasEditControls) {
    return (
      <>
        <ControlButton
          tooltip="New Component"
          onClick={() => onSidebarOpen?.()}
          testId="canvas-add-component-button"
          icon={<Plus className="h-3.5 w-3.5" />}
        />
        <ControlButton
          tooltip="Add Note"
          onClick={() => onAddNote?.()}
          testId="add-note-button"
          icon={<FilePlus className="h-3.5 w-3.5" />}
        />
      </>
    );
  }

  return (
    <>
      <ControlButton
        tooltip="Add component"
        onClick={() => onSidebarOpen?.()}
        testId="open-sidebar-button"
        icon={<Plus className="h-3.5 w-3.5" />}
      />
      <ControlButton
        tooltip="Add Note"
        onClick={() => onAddNote?.()}
        testId="add-note-button"
        icon={<FilePlus className="h-3.5 w-3.5" />}
      />
    </>
  );
}

interface ControlButtonProps {
  tooltip: string;
  onClick: () => void;
  testId: string;
  icon: ReactNode;
  ariaLabel?: string;
}

function ControlButton({ tooltip, onClick, testId, icon, ariaLabel }: ControlButtonProps) {
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <Button
          type="button"
          variant="ghost"
          size="icon-sm"
          onClick={onClick}
          aria-label={ariaLabel ?? tooltip}
          data-testid={testId}
          className={cn(
            "h-7 w-7 shrink-0 rounded-md border-0 shadow-none text-slate-600 hover:bg-slate-100 hover:text-slate-900",
          )}
        >
          {icon}
        </Button>
      </TooltipTrigger>

      <TooltipContent side="left" sideOffset={10} className="max-w-xs">
        {tooltip}
      </TooltipContent>
    </Tooltip>
  );
}
