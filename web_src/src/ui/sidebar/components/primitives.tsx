import * as React from "react";

import { Input } from "../../input";
import { cn } from "@/lib/utils";

const SidebarInset = React.forwardRef<HTMLDivElement, React.ComponentProps<"main">>(({ className, ...props }, ref) => (
  <main
    ref={ref}
    className={cn(
      "bg-background relative flex w-full flex-1 flex-col",
      "md:peer-data-[variant=inset]:m-2 md:peer-data-[variant=inset]:ml-0 md:peer-data-[variant=inset]:rounded-xl md:peer-data-[variant=inset]:shadow md:peer-data-[state=collapsed]:peer-data-[variant=inset]:ml-2",
      className,
    )}
    {...props}
  />
));
SidebarInset.displayName = "SidebarInset";

const SidebarInput = React.forwardRef<React.ElementRef<typeof Input>, React.ComponentProps<typeof Input>>(
  ({ className, ...props }, ref) => (
    <Input
      ref={ref}
      data-sidebar="input"
      className={cn(
        "bg-background focus-visible:ring-sidebar-ring h-8 w-full shadow-none focus-visible:ring-2",
        className,
      )}
      {...props}
    />
  ),
);
SidebarInput.displayName = "SidebarInput";

const SidebarContent = React.forwardRef<HTMLDivElement, React.ComponentProps<"div">>(({ className, ...props }, ref) => (
  <div
    ref={ref}
    data-sidebar="content"
    className={cn(
      "flex min-h-0 flex-1 flex-col gap-2 overflow-auto group-data-[collapsible=icon]:overflow-hidden",
      className,
    )}
    {...props}
  />
));
SidebarContent.displayName = "SidebarContent";

export { SidebarContent, SidebarInput, SidebarInset };
