import { cn } from "@/lib/utils";

import { consoleBadgeClassName } from "./consoleBadgeStyles";

export function ConsoleBadge({
  label,
  className,
  colorClass,
}: {
  label: string;
  className?: string;
  colorClass?: string;
}) {
  if (!label.trim()) return null;
  return <span className={cn(consoleBadgeClassName(label, colorClass), className)}>{label}</span>;
}
