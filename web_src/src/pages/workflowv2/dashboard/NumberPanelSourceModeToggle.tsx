import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";

import { convertNumberPanelMode, detectMode, type NumberSourceMode } from "./numberPanelSourceMode";
import { type NumberPanelContent } from "./panelTypes";

export function NumberPanelSourceModeToggle({
  value,
  onChange,
}: {
  value: NumberPanelContent;
  onChange: (next: NumberPanelContent) => void;
}) {
  const mode = detectMode(value);

  return (
    <div className="space-y-1.5">
      <Label className="text-xs font-medium text-slate-600">Source mode</Label>
      <div className="flex flex-wrap gap-1">
        <Button
          type="button"
          size="sm"
          variant={mode === "single" ? "secondary" : "outline"}
          onClick={() => switchTo("single", mode, value, onChange)}
          data-testid="number-mode-simple"
        >
          Single source
        </Button>
        <Button
          type="button"
          size="sm"
          variant={mode === "composite" ? "secondary" : "outline"}
          onClick={() => switchTo("composite", mode, value, onChange)}
          data-testid="number-mode-composite"
        >
          Multiple memory sources
        </Button>
        <Button
          type="button"
          size="sm"
          variant={mode === "multi" ? "secondary" : "outline"}
          onClick={() => switchTo("multi", mode, value, onChange)}
          data-testid="number-mode-multi"
        >
          Multiple numbers
        </Button>
      </div>
    </div>
  );
}

function switchTo(
  target: NumberSourceMode,
  current: NumberSourceMode,
  value: NumberPanelContent,
  onChange: (next: NumberPanelContent) => void,
): void {
  if (target === current) return;
  onChange(convertNumberPanelMode(target, value));
}
