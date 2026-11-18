import { Button } from "@/components/ui/button";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/ui/dropdownMenu";
import { MoreHorizontal, Pause, Play } from "lucide-react";

export function PlanningSettingsMenu({
  enabled,
  disabled,
  onEnabledChange,
  title = "Planning",
  noun = "planning",
  testId = "planning-settings",
  onDelete,
}: {
  enabled: boolean;
  disabled: boolean;
  onEnabledChange: (enabled: boolean) => void;
  title?: string;
  noun?: string;
  testId?: string;
  onDelete?: () => void;
}) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          variant="ghost"
          size="icon"
          className="size-6"
          disabled={disabled}
          aria-label={`${title} menu`}
          data-testid={`${testId}-menu`}
        >
          <MoreHorizontal className="size-4" aria-hidden />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuItem
          disabled={disabled}
          onSelect={() => onEnabledChange(!enabled)}
          data-testid={`${testId}-${enabled ? "pause" : "resume"}`}
        >
          {enabled ? <Pause aria-hidden /> : <Play aria-hidden />}
          {enabled ? `Pause ${noun}` : `Resume ${noun}`}
        </DropdownMenuItem>
        {onDelete ? (
          <DropdownMenuItem className="text-destructive focus:text-destructive" disabled={disabled} onSelect={onDelete}>
            Delete automation
          </DropdownMenuItem>
        ) : null}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
