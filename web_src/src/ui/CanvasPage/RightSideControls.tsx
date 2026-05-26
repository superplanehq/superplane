import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { Plus, StickyNote } from "lucide-react";
import { memo, type ReactNode } from "react";

export type RightSideControlsProps = {
  mode: "live" | "edit";
  /** When true, only the floating Add note control is shown (component/YAML live in the header while editing). */
  addNoteOnly?: boolean;

  onSidebarOpen: () => void;
  onAddNote: () => void | Promise<void>;
};

export const RightSideControls = memo(function RightSideControls(props: RightSideControlsProps) {
  if (props.mode === "live") return null;
  return (
    <div className="absolute top-4 right-4 z-10 flex flex-col gap-2.5">
      <EditCanvasButtons {...props} />
    </div>
  );
});

function EditCanvasButtons({ addNoteOnly, onSidebarOpen, onAddNote }: RightSideControlsProps) {
  if (addNoteOnly) {
    return <ControlButton tooltip="Add note" onClick={onAddNote} testId="add-note-button" icon={<StickyNote />} />;
  }

  return (
    <>
      <ControlButton tooltip="Add component" onClick={onSidebarOpen} testId="open-sidebar-button" icon={<Plus />} />
      <ControlButton tooltip="Add note" onClick={onAddNote} testId="add-note-button" icon={<StickyNote />} />
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
