import clsx from "clsx";

export interface MaterialSymbolProps {
  /** The name of the Material Symbol (e.g., 'home', 'settings', 'person') */
  name: string;
  /** Size variant */
  size?: "sm" | "md" | "lg" | "xl" | "4xl";
  /** Fill variant (0 = outlined, 1 = filled) */
  fill?: 0 | 1;
  /** Weight variant (100-700) */
  weight?: 100 | 200 | 300 | 400 | 500 | 600 | 700;
  /** Grade variant (-25 to 200) */
  grade?: number;
  /** Optical size (20-48) */
  opticalSize?: number;
  /** Additional CSS classes */
  className?: string;
  /** Data slot attribute for button styling */
  "data-slot"?: string;
}

export function MaterialSymbol({
  name,
  size = "md",
  fill = 0,
  weight = 400,
  grade = 0,
  opticalSize = 24,
  className,
  "data-slot": dataSlot,
}: MaterialSymbolProps) {
  const sizeClasses = {
    sm: "!text-sm", // 14px
    md: "!text-base", // 16px
    lg: "!text-xl", // 20px
    xl: "!text-2xl", // 24px
    "4xl": "!text-4xl", // 32px
  };

  const style = {
    fontVariationSettings: `'FILL' ${fill}, 'wght' ${weight}, 'GRAD' ${grade}, 'opsz' ${opticalSize}`,
  };

  return (
    <span
      className={clsx("material-symbols-outlined select-none", sizeClasses[size], className)}
      style={style}
      aria-hidden="true"
      data-slot={dataSlot}
    >
      {name}
    </span>
  );
}

// Preset variants for common use cases
export const MaterialSymbolFilled = (props: Omit<MaterialSymbolProps, "fill">) => (
  <MaterialSymbol {...props} fill={1} />
);

export const MaterialSymbolLight = (props: Omit<MaterialSymbolProps, "weight">) => (
  <MaterialSymbol {...props} weight={300} />
);

export const MaterialSymbolBold = (props: Omit<MaterialSymbolProps, "weight">) => (
  <MaterialSymbol {...props} weight={600} />
);
