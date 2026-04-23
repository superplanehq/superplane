import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { Code2, Database, FileText, Plus, StickyNote } from "lucide-react";
import type { ReactNode } from "react";

export type RightSideControlsProps = {
  mode: "live" | "edit";

  onSidebarOpen: () => void;
  onAddNote: () => void | Promise<void>;
  onYamlOpen: () => void;
  onMemoryOpen: () => void;
  onReadmeOpen?: () => void;
  memoryItemCount?: number;
};

export function RightSideControls(props: RightSideControlsProps) {
  return (
    <div className="absolute top-4 right-4 z-10 flex flex-col gap-1.5">
      {props.mode === "live" ? <LiveCanvasButtons {...props} /> : <EditCanvasButtons {...props} />}
    </div>
  );
}

function LiveCanvasButtons({ onMemoryOpen, onReadmeOpen }: RightSideControlsProps) {
  return (
    <>
      <ControlButton tooltip="Canvas memory" onClick={onMemoryOpen} testId="open-memory-button" icon={<Database />} />
      {onReadmeOpen ? (
        <ControlButton
          tooltip="Canvas readme"
          onClick={onReadmeOpen}
          testId="open-readme-button"
          icon={<FileText />}
        />
      ) : null}
    </>
  );
}

function EditCanvasButtons({ onSidebarOpen, onAddNote, onYamlOpen, onReadmeOpen }: RightSideControlsProps) {
  return (
    <>
      <ControlButton tooltip="Add component" onClick={onSidebarOpen} testId="open-sidebar-button" icon={<Plus />} />
      <ControlButton tooltip="Add note" onClick={onAddNote} testId="add-note-button" icon={<StickyNote />} />
      <ControlButton tooltip="YAML" onClick={onYamlOpen} testId="open-yaml-modal-button" icon={<Code2 />} />
      {onReadmeOpen ? (
        <ControlButton
          tooltip="Canvas readme"
          onClick={onReadmeOpen}
          testId="open-readme-button"
          icon={<FileText />}
        />
      ) : null}
    </>
  );
}

interface ControlButtonProps {
  tooltip: string;
  onClick: () => void;
  testId: string;
  icon: ReactNode;
}

function ControlButton(props: ControlButtonProps) {
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <Button
          type="button"
          variant="outline"
          onClick={props.onClick}
          aria-label={props.tooltip}
          data-testid={props.testId}
          children={props.icon}
        />
      </TooltipTrigger>

      <TooltipContent side="left" sideOffset={10}>
        {props.tooltip}
      </TooltipContent>
    </Tooltip>
  );
}
