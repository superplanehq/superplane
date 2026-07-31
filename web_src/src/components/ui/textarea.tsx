import * as React from "react";

import { cn } from "@/lib/utils";

function Textarea({ className, ...props }: React.ComponentProps<"textarea">) {
  return (
    <textarea
      data-slot="textarea"
      className={cn(
        "border-edge-strong bg-surface-raised text-content-primary placeholder:text-content-muted flex field-sizing-content min-h-16 w-full rounded-md border px-3 py-2 text-sm wrap-anywhere whitespace-pre-wrap shadow-xs transition-[color,box-shadow] outline-none disabled:cursor-not-allowed disabled:opacity-50",
        "focus:border-focus-ring focus:shadow-none focus:ring-0",
        "aria-invalid:ring-status-danger/20 aria-invalid:border-status-danger",
        className,
      )}
      {...props}
    />
  );
}

export { Textarea };
