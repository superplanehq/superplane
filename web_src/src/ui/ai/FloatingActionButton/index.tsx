import { BotMessageSquare } from "lucide-react";
import { cn } from "../../../lib/utils";

// Hoist static objects outside component to avoid re-creation on every render
const POSITION_CLASSES = {
  "bottom-right": "bottom-6 right-6",
  "bottom-left": "bottom-6 left-6",
  "top-right": "top-6 right-6",
  "top-left": "top-6 left-6",
} as const;

const VARIANT_CLASSES = {
  primary: "bg-stone-900",
  secondary: "border border-stone-300 bg-white/70 shadow-md",
} as const;

export namespace FloatingActionButton {
  export interface Props {
    /**
     * Click handler
     */
    onClick: () => void;

    /**
     * Button label for accessibility and tooltip
     */
    label: string;

    /**
     * Optional text to display next to the icon
     */
    text?: string;

    /**
     * Visual style variant
     */
    variant?: "primary" | "secondary";

    /**
     * Size of the FAB
     */
    size?: "normal" | "large";

    /**
     * Position on screen
     */
    position?: "bottom-right" | "bottom-left" | "top-right" | "top-left";

    /**
     * Whether the FAB is disabled
     */
    disabled?: boolean;

    /**
     * Custom className
     */
    className?: string;

    /**
     * Whether to show a tooltip on hover
     */
    showTooltip?: boolean;

    /**
     * Custom z-index value
     */
    zIndex?: number;
  }
}

export function FloatingActionButton({
  onClick,
  label,
  text,
  variant = "primary",
  size = "normal",
  position = "bottom-right",
  disabled = false,
  className,
  showTooltip = true,
  zIndex = 50,
}: FloatingActionButton.Props) {
  const buttonClasses = cn(
    "fixed flex items-center justify-center transition-all duration-200 ease-in-out",
    "focus:outline-none",
    "disabled:opacity-50 disabled:cursor-not-allowed disabled:hover:shadow-md",
    "rounded-lg p-2 px-4",
    "gap-2 hover:gap-3",
    POSITION_CLASSES[position],
    VARIANT_CLASSES[variant],
    className
  );

  return (
    <button
      onClick={onClick}
      disabled={disabled}
      className={buttonClasses}
      aria-label={label}
      title={showTooltip ? label : undefined}
      style={{ zIndex }}
    >
      <BotMessageSquare className="text-white" size={16} />

      <span className="font-medium whitespace-nowrap text-white text-sm">
        AI Assistant
      </span>
    </button>
  );
}
