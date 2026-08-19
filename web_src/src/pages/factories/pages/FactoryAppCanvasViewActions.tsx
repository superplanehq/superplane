import { ChevronDown } from "lucide-react";

import { Button } from "@/components/ui/button";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/ui/dropdownMenu";

type FactoryAppCanvasViewActionsProps = {
  yamlDisabled?: boolean;
  onOpenVisualEditor: () => void;
  onAskAgent: () => void;
  onOpenDesktopAgentSetup: () => void;
  onViewYaml: () => void;
};

export function FactoryAppCanvasViewActions({
  yamlDisabled = false,
  onOpenVisualEditor,
  onAskAgent,
  onOpenDesktopAgentSetup,
  onViewYaml,
}: FactoryAppCanvasViewActionsProps) {
  return (
    <div className="flex shrink-0 items-center gap-2 pt-0.5">
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button type="button" size="sm" data-testid="factory-app-edit-menu">
            Edit
            <ChevronDown className="size-3.5" aria-hidden />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" className="w-56">
          <DropdownMenuItem onSelect={onOpenVisualEditor} data-testid="factory-app-edit-visual">
            Open visual editor
          </DropdownMenuItem>
          <DropdownMenuItem onSelect={onAskAgent} data-testid="factory-app-edit-ask-agent">
            Ask agent
          </DropdownMenuItem>
          <DropdownMenuItem onSelect={onOpenDesktopAgentSetup} data-testid="factory-app-edit-desktop-agent">
            Set up desktop agent
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
      <Button
        type="button"
        variant="outline"
        size="sm"
        disabled={yamlDisabled}
        onClick={onViewYaml}
        data-testid="factory-app-view-yaml"
      >
        View YAML
      </Button>
    </div>
  );
}
