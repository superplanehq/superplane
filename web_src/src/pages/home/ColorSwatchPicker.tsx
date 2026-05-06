import { Check } from "lucide-react";
import { CANVAS_GROUP_COLORS, type CanvasGroupColor } from "../../hooks/useCanvasData";
import { cn } from "../../lib/utils";
import { GROUP_SWATCH_CLASSES, colorLabel } from "./shared";

interface ColorSwatchPickerProps {
  selectedColor: CanvasGroupColor;
  onSelect: (color: CanvasGroupColor) => void;
  size?: "sm" | "md";
  isColorDisabled?: (color: CanvasGroupColor) => boolean;
}

export const ColorSwatchPicker = ({
  selectedColor,
  onSelect,
  size = "md",
  isColorDisabled,
}: ColorSwatchPickerProps) => {
  const dimensions = size === "sm" ? "h-5 w-5" : "h-6 w-6";

  return (
    <div className="flex items-center gap-2">
      {CANVAS_GROUP_COLORS.map((color) => {
        const isSelected = selectedColor === color;
        return (
          <button
            key={color}
            type="button"
            aria-label={`${colorLabel(color)} group color`}
            className={cn(
              "flex items-center justify-center rounded-full border border-slate-950/15 text-white",
              dimensions,
              GROUP_SWATCH_CLASSES[color],
              isSelected && "ring-2 ring-gray-900 ring-offset-1",
            )}
            onClick={() => onSelect(color)}
            disabled={isColorDisabled?.(color) ?? false}
          >
            {isSelected ? <Check className="h-3 w-3" /> : null}
          </button>
        );
      })}
    </div>
  );
};
