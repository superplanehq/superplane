import * as React from "react";

import { cn } from "@/lib/utils";

function Textarea({ className, ...props }: React.ComponentProps<"textarea">) {
  return (
    <textarea
      data-slot="textarea"
      className={cn(
        "flex field-sizing-content min-h-16 w-full rounded-md border border-gray-300 bg-white px-3 py-2 text-sm wrap-anywhere whitespace-pre-wrap text-[rgba(10,10,10,1)] shadow-xs transition-[color,box-shadow] outline-none placeholder:text-gray-500 disabled:cursor-not-allowed disabled:opacity-50 dark:border-gray-600/70 dark:bg-gray-800 dark:text-gray-100 dark:placeholder:text-gray-500",
        "focus:border-gray-500 focus:shadow-none focus:ring-0 dark:focus:border-gray-500",
        "aria-invalid:ring-destructive/20 dark:aria-invalid:ring-destructive/40 aria-invalid:border-destructive",
        className,
      )}
      {...props}
    />
  );
}

export { Textarea };
