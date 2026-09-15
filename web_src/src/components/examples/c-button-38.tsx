import type { ComponentProps, ReactNode } from "react";

import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

type CountButtonProps = ComponentProps<typeof Button> & {
  count: ReactNode;
  countClassName?: string;
};

/** ReUI c-button-38: outline button with a trailing count. */
export function CountButton({ children, count, countClassName, className, ...props }: CountButtonProps) {
  return (
    <Button className={cn("pe-0", className)} variant="outline" {...props}>
      {children}
      <span
        className={cn(
          "text-muted-foreground before:bg-border relative ms-1 px-2 text-xs font-medium before:absolute before:inset-0 before:left-0 before:w-px",
          countClassName,
        )}
      >
        {count}
      </span>
    </Button>
  );
}
