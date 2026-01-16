import React, { forwardRef } from "react";

import { cn } from "@/lib/utils";

export const Textarea = forwardRef(function Textarea(
  {
    className,
    resizable = true,
    ...props
  }: { className?: string; resizable?: boolean } & React.ComponentPropsWithoutRef<"textarea">,
  ref: React.ForwardedRef<HTMLTextAreaElement>,
) {
  return (
    <textarea
      ref={ref}
      data-slot="textarea"
      className={cn(
        "border-input placeholder:text-muted-foreground focus-visible:border-gray-500 aria-invalid:ring-destructive/20 dark:aria-invalid:ring-destructive/40 aria-invalid:border-destructive dark:bg-input/30 flex field-sizing-content min-h-16 w-full rounded-md border bg-transparent px-3 py-2 text-sm shadow-xs transition-[color,box-shadow] outline-none disabled:cursor-not-allowed disabled:opacity-50 text-[rgba(10,10,10,1)]",
        resizable ? "resize-y" : "resize-none",
        className,
      )}
      {...props}
    />
  );
});
