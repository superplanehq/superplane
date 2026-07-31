import * as React from "react";

import { cn } from "@/lib/utils";

function Input({ className, type, ...props }: React.ComponentProps<"input">) {
  return (
    <input
      type={type}
      data-slot="input"
      className={cn(
        "font-sm border-edge-strong bg-surface-raised text-content-primary file:text-content-primary placeholder:text-content-muted h-8 w-full min-w-0 rounded-md border px-3 py-1 text-sm transition-[color,box-shadow] outline-none file:inline-flex file:h-7 file:border-0 file:bg-transparent file:text-sm file:font-medium disabled:pointer-events-none disabled:cursor-not-allowed disabled:opacity-50",
        "focus:border-focus-ring focus:shadow-none focus:ring-0",
        "aria-invalid:ring-status-danger/20 aria-invalid:border-status-danger",
        className,
      )}
      {...props}
    />
  );
}

export { Input };
