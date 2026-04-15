import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { Plus, StickyNote } from "lucide-react";

export type RightSideControlsProps = {
  readOnly: boolean;
  onSidebarOpen: () => void;
  onAddNote: () => void | Promise<void>;
};

export function RightSideControls({ readOnly, onSidebarOpen, onAddNote }: RightSideControlsProps) {
  return (
    <div className="absolute top-4 right-4 z-10 flex flex-col gap-1.5">
      <ControlButton
        tooltip="Add component"
        hidden={readOnly}
        onClick={onSidebarOpen}
        testId="open-sidebar-button"
        icon={<Plus />}
      />

      <ControlButton
        tooltip="Add note"
        hidden={readOnly}
        onClick={onAddNote}
        testId="add-note-button"
        icon={<StickyNote />}
      />
    </div>
  );
}

interface ControlButtonProps {
  hidden: boolean;
  tooltip: string;
  onClick: () => void;
  testId?: string;
  icon: React.ReactNode;
}

function ControlButton(props: ControlButtonProps) {
  if (props.hidden) return null;

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <Button
          type="button"
          variant="outline"
          onClick={props.onClick}
          aria-label={props.tooltip}
          data-testid={props.testId}
        >
          {props.icon}
        </Button>
      </TooltipTrigger>

      <TooltipContent side="left" sideOffset={10}>
        {props.tooltip}
      </TooltipContent>
    </Tooltip>
  );
}
