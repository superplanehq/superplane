import * as React from "react";

import { Separator } from "../../separator";
import { cn } from "@/lib/utils";

const SidebarSeparator = React.forwardRef<React.ElementRef<typeof Separator>, React.ComponentProps<typeof Separator>>(
  ({ className, ...props }, ref) => (
    <Separator
      ref={ref}
      data-sidebar="separator"
      className={cn("bg-sidebar-border mx-2 w-auto", className)}
      {...props}
    />
  ),
);
SidebarSeparator.displayName = "SidebarSeparator";

export { SidebarSeparator };
