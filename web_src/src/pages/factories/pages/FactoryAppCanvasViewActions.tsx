import { Button } from "@/components/ui/button";

type FactoryAppCanvasViewActionsProps = {
  onOpenVisualEditor: () => void;
};

export function FactoryAppCanvasViewActions({ onOpenVisualEditor }: FactoryAppCanvasViewActionsProps) {
  return (
    <div className="flex items-start justify-end">
      <Button type="button" size="sm" onClick={onOpenVisualEditor} data-testid="factory-app-edit">
        Edit
      </Button>
    </div>
  );
}
