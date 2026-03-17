import * as React from "react";
import * as DialogPrimitive from "@radix-ui/react-dialog";
import { XIcon } from "lucide-react";
import { cn } from "@/lib/utils";

function ResizableDialog({ ...props }: React.ComponentProps<typeof DialogPrimitive.Root>) {
  return <DialogPrimitive.Root data-slot="resizable-dialog" {...props} />;
}

function ResizableDialogTrigger({ ...props }: React.ComponentProps<typeof DialogPrimitive.Trigger>) {
  return <DialogPrimitive.Trigger data-slot="resizable-dialog-trigger" {...props} />;
}

function ResizableDialogPortal({ ...props }: React.ComponentProps<typeof DialogPrimitive.Portal>) {
  return <DialogPrimitive.Portal data-slot="resizable-dialog-portal" {...props} />;
}

function ResizableDialogClose({ ...props }: React.ComponentProps<typeof DialogPrimitive.Close>) {
  return <DialogPrimitive.Close data-slot="resizable-dialog-close" {...props} />;
}

function ResizableDialogOverlay({ className, ...props }: React.ComponentProps<typeof DialogPrimitive.Overlay>) {
  return (
    <DialogPrimitive.Overlay
      data-slot="resizable-dialog-overlay"
      className={cn(
        "data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0 fixed inset-0 z-50 bg-black/80",
        className,
      )}
      {...props}
    />
  );
}

interface ResizableDialogContentProps extends React.ComponentProps<typeof DialogPrimitive.Content> {
  showCloseButton?: boolean;
  defaultWidth?: string;
  defaultHeight?: string;
  minWidth?: number;
  minHeight?: number;
  maxWidth?: string;
  maxHeight?: string;
  storageKey?: string;
}

function ResizableDialogContent({
  className,
  children,
  showCloseButton = true,
  defaultWidth = "80vw",
  defaultHeight = "80vh",
  minWidth = 400,
  minHeight = 300,
  maxWidth = "95vw",
  maxHeight = "95vh",
  storageKey,
  ...props
}: ResizableDialogContentProps) {
  const contentRef = React.useRef<HTMLDivElement>(null);
  const [size, setSize] = React.useState<{ width: string; height: string }>(() => {
    // Try to restore size from localStorage if storageKey is provided
    if (storageKey && typeof window !== "undefined") {
      try {
        const stored = localStorage.getItem(`resizable-dialog-${storageKey}`);
        if (stored) {
          const parsed = JSON.parse(stored);
          return { width: parsed.width, height: parsed.height };
        }
      } catch {
        // Ignore errors
      }
    }
    return { width: defaultWidth, height: defaultHeight };
  });

  const [isResizing, setIsResizing] = React.useState(false);
  const resizeStartRef = React.useRef<{
    x: number;
    y: number;
    width: number;
    height: number;
    handle: string;
  } | null>(null);

  // Save size to localStorage when it changes
  React.useEffect(() => {
    if (storageKey && typeof window !== "undefined") {
      try {
        localStorage.setItem(`resizable-dialog-${storageKey}`, JSON.stringify(size));
      } catch {
        // Ignore errors
      }
    }
  }, [size, storageKey]);

  const handleMouseDown = React.useCallback((e: React.MouseEvent, handle: string) => {
    e.preventDefault();
    e.stopPropagation();

    const rect = contentRef.current?.getBoundingClientRect();
    if (!rect) return;

    resizeStartRef.current = {
      x: e.clientX,
      y: e.clientY,
      width: rect.width,
      height: rect.height,
      handle,
    };
    setIsResizing(true);
  }, []);

  React.useEffect(() => {
    if (!isResizing || !resizeStartRef.current) return;

    const handleMouseMove = (e: MouseEvent) => {
      if (!resizeStartRef.current) return;

      const { x, y, width, height, handle } = resizeStartRef.current;
      const deltaX = e.clientX - x;
      const deltaY = e.clientY - y;

      let newWidth = width;
      let newHeight = height;

      // Handle different resize directions
      if (handle.includes("right")) {
        newWidth = Math.max(minWidth, width + deltaX);
      }
      if (handle.includes("left")) {
        newWidth = Math.max(minWidth, width - deltaX);
      }
      if (handle.includes("bottom")) {
        newHeight = Math.max(minHeight, height + deltaY);
      }
      if (handle.includes("top")) {
        newHeight = Math.max(minHeight, height - deltaY);
      }

      // Apply max constraints
      const maxWidthPx = maxWidth.includes("vw")
        ? (parseFloat(maxWidth) / 100) * window.innerWidth
        : parseFloat(maxWidth);
      const maxHeightPx = maxHeight.includes("vh")
        ? (parseFloat(maxHeight) / 100) * window.innerHeight
        : parseFloat(maxHeight);

      newWidth = Math.min(newWidth, maxWidthPx);
      newHeight = Math.min(newHeight, maxHeightPx);

      setSize({ width: `${newWidth}px`, height: `${newHeight}px` });
    };

    const handleMouseUp = () => {
      setIsResizing(false);
      resizeStartRef.current = null;
    };

    document.addEventListener("mousemove", handleMouseMove);
    document.addEventListener("mouseup", handleMouseUp);

    return () => {
      document.removeEventListener("mousemove", handleMouseMove);
      document.removeEventListener("mouseup", handleMouseUp);
    };
  }, [isResizing, minWidth, minHeight, maxWidth, maxHeight]);

  return (
    <ResizableDialogPortal>
      <ResizableDialogOverlay />
      <DialogPrimitive.Content
        ref={contentRef}
        data-slot="resizable-dialog-content"
        className={cn(
          "bg-background data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0 data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95 fixed top-[50%] left-[50%] z-50 grid translate-x-[-50%] translate-y-[-50%] gap-4 rounded-lg p-6 shadow-lg duration-200",
          isResizing && "select-none",
          className,
        )}
        style={{
          width: size.width,
          height: size.height,
          maxWidth,
          maxHeight,
        }}
        {...props}
      >
        {children}

        {/* Resize handles */}
        <div
          className="absolute top-0 right-0 bottom-0 w-1 cursor-ew-resize hover:bg-blue-500/20 active:bg-blue-500/40 transition-colors"
          onMouseDown={(e) => handleMouseDown(e, "right")}
        />
        <div
          className="absolute top-0 left-0 bottom-0 w-1 cursor-ew-resize hover:bg-blue-500/20 active:bg-blue-500/40 transition-colors"
          onMouseDown={(e) => handleMouseDown(e, "left")}
        />
        <div
          className="absolute left-0 right-0 bottom-0 h-1 cursor-ns-resize hover:bg-blue-500/20 active:bg-blue-500/40 transition-colors"
          onMouseDown={(e) => handleMouseDown(e, "bottom")}
        />
        <div
          className="absolute left-0 right-0 top-0 h-1 cursor-ns-resize hover:bg-blue-500/20 active:bg-blue-500/40 transition-colors"
          onMouseDown={(e) => handleMouseDown(e, "top")}
        />

        {/* Corner handles */}
        <div
          className="absolute top-0 right-0 w-4 h-4 cursor-nesw-resize hover:bg-blue-500/20 active:bg-blue-500/40 transition-colors rounded-tr-lg"
          onMouseDown={(e) => handleMouseDown(e, "top-right")}
        />
        <div
          className="absolute top-0 left-0 w-4 h-4 cursor-nwse-resize hover:bg-blue-500/20 active:bg-blue-500/40 transition-colors rounded-tl-lg"
          onMouseDown={(e) => handleMouseDown(e, "top-left")}
        />
        <div
          className="absolute bottom-0 right-0 w-4 h-4 cursor-nwse-resize hover:bg-blue-500/20 active:bg-blue-500/40 transition-colors rounded-br-lg"
          onMouseDown={(e) => handleMouseDown(e, "bottom-right")}
        />
        <div
          className="absolute bottom-0 left-0 w-4 h-4 cursor-nesw-resize hover:bg-blue-500/20 active:bg-blue-500/40 transition-colors rounded-bl-lg"
          onMouseDown={(e) => handleMouseDown(e, "bottom-left")}
        />

        {showCloseButton && (
          <DialogPrimitive.Close
            data-slot="resizable-dialog-close"
            className="ring-offset-background focus:ring-ring data-[state=open]:bg-accent data-[state=open]:text-muted-foreground absolute top-4 right-4 rounded-xs opacity-70 transition-opacity hover:opacity-100 focus:ring-2 focus:ring-offset-2 focus:outline-hidden disabled:pointer-events-none [&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg:not([class*='size-'])]:size-4"
          >
            <XIcon />
            <span className="sr-only">Close</span>
          </DialogPrimitive.Close>
        )}
      </DialogPrimitive.Content>
    </ResizableDialogPortal>
  );
}

function ResizableDialogHeader({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      data-slot="resizable-dialog-header"
      className={cn("flex flex-col space-y-1.5 text-center sm:text-left", className)}
      {...props}
    />
  );
}

function ResizableDialogFooter({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      data-slot="resizable-dialog-footer"
      className={cn("flex flex-col-reverse sm:flex-row sm:justify-end sm:space-x-2", className)}
      {...props}
    />
  );
}

function ResizableDialogTitle({ className, ...props }: React.ComponentProps<typeof DialogPrimitive.Title>) {
  return (
    <DialogPrimitive.Title
      data-slot="resizable-dialog-title"
      className={cn("font-semibold text-lg leading-none tracking-tight", className)}
      {...props}
    />
  );
}

function ResizableDialogDescription({ className, ...props }: React.ComponentProps<typeof DialogPrimitive.Description>) {
  return (
    <DialogPrimitive.Description
      data-slot="resizable-dialog-description"
      className={cn("text-muted-foreground text-sm", className)}
      {...props}
    />
  );
}

export {
  ResizableDialog,
  ResizableDialogPortal,
  ResizableDialogOverlay,
  ResizableDialogClose,
  ResizableDialogTrigger,
  ResizableDialogContent,
  ResizableDialogHeader,
  ResizableDialogFooter,
  ResizableDialogTitle,
  ResizableDialogDescription,
};
