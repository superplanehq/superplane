import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";

import {
  convertNumberPanelMode,
  countNonMemoryMetrics,
  detectMode,
  type NumberSourceMode,
} from "./numberPanelSourceMode";
import { type NumberPanelContent } from "./panelTypes";

export function NumberPanelSourceModeToggle({
  value,
  onChange,
}: {
  value: NumberPanelContent;
  onChange: (next: NumberPanelContent) => void;
}) {
  const mode = detectMode(value);
  // "Multiple memory sources" can only hold memory data, so switching to it
  // from multi-number mode would silently drop any runs/executions numbers.
  // Block the switch (and explain why) instead of losing them on save.
  const droppedByComposite = mode === "multi" ? countNonMemoryMetrics(value) : 0;
  const compositeDisabled = droppedByComposite > 0;
  const compositeDisabledReason = compositeDisabled
    ? `Multiple memory sources only supports memory data. Switching would drop ${droppedByComposite} non-memory number${droppedByComposite === 1 ? "" : "s"} (runs/executions). Convert ${droppedByComposite === 1 ? "it" : "them"} to a memory source or remove ${droppedByComposite === 1 ? "it" : "them"} first.`
    : undefined;

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
        <CompositeModeButton
          active={mode === "composite"}
          disabled={compositeDisabled}
          disabledReason={compositeDisabledReason}
          onClick={() => switchTo("composite", mode, value, onChange)}
        />
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
      {compositeDisabledReason ? (
        <p className="text-[11px] text-amber-700" data-testid="number-mode-composite-warning">
          {compositeDisabledReason}
        </p>
      ) : null}
    </div>
  );
}

function CompositeModeButton({
  active,
  disabled,
  disabledReason,
  onClick,
}: {
  active: boolean;
  disabled: boolean;
  disabledReason?: string;
  onClick: () => void;
}) {
  const button = (
    <Button
      type="button"
      size="sm"
      variant={active ? "secondary" : "outline"}
      // Use aria-disabled + an onClick guard rather than the native disabled
      // attribute so the explanatory tooltip still fires on hover.
      aria-disabled={disabled}
      className={disabled ? "cursor-not-allowed opacity-50" : undefined}
      onClick={() => {
        if (disabled) return;
        onClick();
      }}
      data-testid="number-mode-composite"
    >
      Multiple memory sources
    </Button>
  );

  if (!disabled || !disabledReason) return button;

  return (
    <Tooltip>
      <TooltipTrigger asChild>{button}</TooltipTrigger>
      <TooltipContent className="max-w-xs">{disabledReason}</TooltipContent>
    </Tooltip>
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
