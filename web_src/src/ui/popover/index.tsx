"use client";

import * as React from "react";
import * as PopoverPrimitive from "@radix-ui/react-popover";

import { cn } from "@/lib/utils";

const Popover = PopoverPrimitive.Root;

const PopoverTrigger = PopoverPrimitive.Trigger;

const PopoverAnchor = PopoverPrimitive.Anchor;

const PopoverContent = React.forwardRef<
  React.ElementRef<typeof PopoverPrimitive.Content>,
  React.ComponentPropsWithoutRef<typeof PopoverPrimitive.Content>
>(({ className, align = "center", sideOffset = 4, ...props }, ref) => {
  // #region agent log
  React.useEffect(() => {
    if (ref && "current" in ref && ref.current) {
      const el = ref.current;
      const computed = window.getComputedStyle(el);
      const logData = {
        location: "popover/index.tsx:19",
        message: "PopoverContent rendered with data-slot check",
        data: {
          className: el.className,
          bgColor: computed.backgroundColor,
          border: computed.border,
          borderRadius: computed.borderRadius,
          padding: computed.padding,
          width: computed.width,
          hasDataSlot: el.getAttribute("data-slot"),
          children: Array.from(el.children).map((c) => ({
            tag: c.tagName,
            className: c.className,
            dataSlot: c.getAttribute("data-slot"),
          })),
        },
        timestamp: Date.now(),
        sessionId: "debug-session",
        runId: "post-fix",
        hypothesisId: "E",
      };
      fetch("http://127.0.0.1:7242/ingest/f719ffac-e1c8-4cef-8f17-d4bc91ac736c", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(logData),
      }).catch(() => {});
    }
  }, [ref]);
  // #endregion
  return (
    <PopoverPrimitive.Portal>
      <PopoverPrimitive.Content
        ref={ref}
        align={align}
        sideOffset={sideOffset}
        data-slot="popover-content"
        className={cn(
          "bg-popover text-popover-foreground data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0 data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95 data-[side=bottom]:slide-in-from-top-2 data-[side=left]:slide-in-from-right-2 data-[side=right]:slide-in-from-left-2 data-[side=top]:slide-in-from-bottom-2 z-50 w-72 origin-[--radix-popover-content-transform-origin] rounded-md border p-4 shadow-md outline-none",
          className,
        )}
        {...props}
      />
    </PopoverPrimitive.Portal>
  );
});
PopoverContent.displayName = PopoverPrimitive.Content.displayName;

export { Popover, PopoverAnchor, PopoverContent, PopoverTrigger };
