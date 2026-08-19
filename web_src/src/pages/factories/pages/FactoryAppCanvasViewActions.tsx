import { Button } from "@/components/ui/button";

type FactoryAppCanvasViewActionsProps = {
  onOpenVisualEditor: () => void;
};

export function FactoryAppCanvasViewActions({ onOpenVisualEditor }: FactoryAppCanvasViewActionsProps) {
  return (
    <div className="flex shrink-0 items-center gap-2 pt-0.5">
      <Button type="button" size="sm" onClick={onOpenVisualEditor} data-testid="factory-app-edit">
        Edit
      </Button>
    </div>
  );
}
