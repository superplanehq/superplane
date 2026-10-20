import type { ReactNode } from "react";
import { Label } from "@/components/ui/label";

export function ImplementSettingRow({
  label,
  description,
  htmlFor,
  children,
}: {
  label: string;
  description: string;
  htmlFor?: string;
  children: ReactNode;
}) {
  return (
    <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between sm:gap-8">
      <div className="flex min-w-0 flex-1 flex-col gap-1.5">
        {htmlFor ? (
          <Label htmlFor={htmlFor} className="text-base sm:text-sm">
            {label}
          </Label>
        ) : (
          <p className="text-base font-medium sm:text-sm">{label}</p>
        )}
        <p id={htmlFor ? `${htmlFor}-help` : undefined} className="text-xs text-muted-foreground">
          {description}
        </p>
      </div>
      <div className="flex min-h-8 w-full shrink-0 items-center gap-2 sm:w-64 sm:justify-end">{children}</div>
    </div>
  );
}
