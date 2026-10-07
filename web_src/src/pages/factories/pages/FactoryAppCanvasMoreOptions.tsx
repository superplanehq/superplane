import { MoreHorizontal } from "lucide-react";

import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/ui/dropdownMenu";

type FactoryAppCanvasMoreOptionsProps = {
  onViewYaml: () => void;
  onEditWithLocalAgent: () => void;
  /** Omitted (or undefined) hides the item — no bundled template matches this app. */
  onResetToFactoryDefaults?: () => void;
};

export function FactoryAppCanvasMoreOptions({
  onViewYaml,
  onEditWithLocalAgent,
  onResetToFactoryDefaults,
}: FactoryAppCanvasMoreOptionsProps) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          type="button"
          variant="ghost"
          size="sm"
          className="h-7 px-2.5 text-[12px] font-medium text-muted-foreground"
          aria-label="More options"
          data-testid="factory-app-more-options"
        >
          <MoreHorizontal className="size-3.5" aria-hidden />
          More options
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-52">
        <DropdownMenuItem onSelect={onViewYaml} data-testid="factory-app-view-yaml">
          View YAML
        </DropdownMenuItem>
        <DropdownMenuItem onSelect={onEditWithLocalAgent} data-testid="factory-app-edit-local-agent">
          Edit with a local agent
        </DropdownMenuItem>
        {onResetToFactoryDefaults ? (
          <>
            <DropdownMenuSeparator />
            <DropdownMenuItem onSelect={onResetToFactoryDefaults} data-testid="factory-app-reset-defaults">
              Reset to factory defaults
            </DropdownMenuItem>
          </>
        ) : null}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
