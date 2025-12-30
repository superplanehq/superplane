import { resolveIcon } from "@/lib/utils";
import clsx from "clsx";

export interface IconProps {
  /** The name of the icon (Material Symbol name) */
  name: string;
  /** Size variant */
  size?: "sm" | "md" | "lg" | "xl" | "4xl";
  /** Additional CSS classes */
  className?: string;
  /** Data slot attribute for button styling */
  "data-slot"?: string;
}

export function Icon({ name, size = "md", className, "data-slot": dataSlot }: IconProps) {
  const IconComponent = resolveIcon(name);

  if (!IconComponent) {
    console.warn(`Icon "${name}" not found in icon mapping`);
    return null;
  }

  const sizeClasses = {
    sm: "h-4 w-4", // 16px
    md: "h-4 w-4", // 16px
    lg: "h-5 w-5", // 20px
    xl: "h-6 w-6", // 24px
    "4xl": "h-8 w-8", // 32px
  };

  return (
    <IconComponent
      className={clsx("flex-shrink-0", sizeClasses[size], className)}
      aria-hidden={true}
      data-slot={dataSlot}
    />
  );
}

// Export for backward compatibility during migration
export const MaterialSymbol = Icon;
export const MaterialSymbolFilled = Icon;
export const MaterialSymbolLight = Icon;
export const MaterialSymbolBold = Icon;
