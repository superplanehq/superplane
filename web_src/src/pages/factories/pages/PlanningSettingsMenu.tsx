import { Button } from "@/components/ui/button";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/ui/dropdownMenu";
import { MoreHorizontal, Pause, Play } from "lucide-react";

export function PlanningSettingsMenu({
  enabled,
  disabled,
  onEnabledChange,
}: {
  enabled: boolean;
  disabled: boolean;
  onEnabledChange: (enabled: boolean) => void;
}) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          variant="ghost"
          size="icon"
          className="size-6"
          disabled={disabled}
          aria-label="Planning menu"
          data-testid="planning-settings-menu"
        >
          <MoreHorizontal className="size-4" aria-hidden />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuItem
          disabled={disabled}
          onSelect={() => onEnabledChange(!enabled)}
          data-testid={enabled ? "planning-settings-pause" : "planning-settings-resume"}
        >
          {enabled ? <Pause aria-hidden /> : <Play aria-hidden />}
          {enabled ? "Pause planning" : "Resume planning"}
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
