import { CANVAS_FOLDER_COLORS, type CanvasFolderColor } from "@/hooks/useCanvasData";
import { cn } from "@/lib/utils";
import { Check } from "lucide-react";
import { FOLDER_COLOR_OPTIONS } from "./canvasFolderStyles";

interface CanvasFolderColorPickerProps {
  selectedColor: CanvasFolderColor;
  onColorChange: (color: CanvasFolderColor) => void;
  isColorDisabled?: (color: CanvasFolderColor) => boolean;
  size?: "sm" | "md";
  className?: string;
}

export function CanvasFolderColorPicker({
  selectedColor,
  onColorChange,
  isColorDisabled,
  size = "sm",
  className,
}: CanvasFolderColorPickerProps) {
  const sizeClassName = size === "md" ? "h-6 w-6" : "h-5 w-5";

  return (
    <div className={cn("flex items-center gap-2", className)}>
      {CANVAS_FOLDER_COLORS.map((color) => (
        <button
          key={color}
          type="button"
          aria-label={`${FOLDER_COLOR_OPTIONS[color].label} folder color`}
          className={cn(
            "flex items-center justify-center rounded-full border border-slate-950/15 dark:border-gray-700/70",
            sizeClassName,
            FOLDER_COLOR_OPTIONS[color].swatchClass,
            FOLDER_COLOR_OPTIONS[color].swatchForegroundClass,
            selectedColor === color &&
              "ring-2 ring-gray-900 ring-offset-1 dark:ring-gray-300 dark:ring-offset-gray-900",
          )}
          onClick={() => onColorChange(color)}
          disabled={isColorDisabled?.(color) ?? false}
        >
          {selectedColor === color ? <Check className="h-3 w-3" /> : null}
        </button>
      ))}
    </div>
  );
}
