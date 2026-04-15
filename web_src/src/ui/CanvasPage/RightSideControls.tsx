import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { Code2, Plus, StickyNote } from "lucide-react";
import type { ReactNode } from "react";

export type RightSideControlsProps = {
  readOnly: boolean;
  onSidebarOpen: () => void;
  onAddNote: () => void | Promise<void>;
  onYamlOpen: () => void;
};

export function RightSideControls({ readOnly, onSidebarOpen, onAddNote, onYamlOpen }: RightSideControlsProps) {
  if (readOnly) {
    return null;
  }

  return (
    <div className="absolute top-4 right-4 z-10 flex flex-col gap-1.5">
      <ControlButton tooltip="Add component" onClick={onSidebarOpen} testId="open-sidebar-button" icon={<Plus />} />
      <ControlButton tooltip="Add note" onClick={onAddNote} testId="add-note-button" icon={<StickyNote />} />
      <ControlButton tooltip="YAML" onClick={onYamlOpen} testId="open-yaml-modal-button" icon={<Code2 />} />
    </div>
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
