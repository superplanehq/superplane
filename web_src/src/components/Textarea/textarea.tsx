import clsx from "clsx";
import React, { forwardRef } from "react";

export const Textarea = forwardRef(function Textarea(
  {
    className,
    resizable = true,
    ...props
  }: { className?: string; resizable?: boolean } & React.ComponentPropsWithoutRef<"textarea">,
  ref: React.ForwardedRef<HTMLTextAreaElement>,
) {
  return (
    <span
      data-slot="control"
      className={clsx([
        className,
        "relative block w-full",
        "before:absolute before:inset-px before:rounded-[calc(var(--radius-lg)-1px)] before:bg-white before:shadow-sm",
        "dark:before:hidden",
        "after:pointer-events-none after:absolute after:inset-0 after:rounded-lg after:ring-transparent after:ring-inset sm:focus-within:after:ring-2 sm:focus-within:after:ring-blue-500",
        "has-data-disabled:opacity-50 has-data-disabled:before:bg-zinc-950/5 has-data-disabled:before:shadow-none",
      ])}
    >
      <textarea
        ref={ref}
        {...props}
        className={clsx([
          "relative block h-full w-full appearance-none rounded-lg px-3 py-2 sm:px-3 sm:py-1.5",
          "text-base/6 text-zinc-950 placeholder:text-zinc-500 sm:text-sm/6 dark:text-white",
          "border border-zinc-950/10 hover:border-zinc-950/20 dark:border-white/10 dark:hover:border-white/20",
          "bg-transparent dark:bg-white/5",
          "focus:outline-none",
          "disabled:border-zinc-950/20 dark:disabled:border-white/15 dark:disabled:bg-white/2.5",
          resizable ? "resize-y" : "resize-none",
        ])}
      />
    </span>
  );
});
