import * as React from "react";

import { cn } from "@/lib/utils";

function Input({ className, type, ...props }: React.ComponentProps<"input">) {
  return (
    <input
      type={type}
      data-slot="input"
      className={cn(
        "font-sm bg-white dark:bg-gray-800 border-gray-300 dark:border-gray-600 shadow-xs file:text-foreground placeholder:text-gray-500 dark:placeholder:text-gray-400 selection:bg-primary selection:text-primary-foreground h-8 w-full min-w-0 rounded-md border px-3 py-1 text-sm transition-[color,box-shadow] outline-none file:inline-flex file:h-7 file:border-0 file:bg-transparent file:text-sm file:font-medium disabled:pointer-events-none disabled:cursor-not-allowed disabled:opacity-50 text-[rgba(10,10,10,1)] dark:text-gray-100",
        "focus-visible:border-gray-500 focus-visible:ring-ring/50",
        "aria-invalid:ring-destructive/20 dark:aria-invalid:ring-destructive/40 aria-invalid:border-destructive",
        className,
      )}
      {...props}
    />
  );
}

export { Input };
