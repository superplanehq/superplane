import { useEffect, useRef, useState } from "react";

import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { showErrorToast } from "@/lib/toast";
import { cn } from "@/lib/utils";

/** How long the "Copied" confirmation stays visible. */
const COPIED_CONFIRMATION_MS = 1600;

interface CopyableKeyButtonProps {
  value: string;
  className?: string;
  testId?: string;
}

/**
 * Compact task ID that copies on click. Use next to a title so the name
 * stays primary and the ID stays available without a kicker line.
 */
export function CopyableKeyButton({
  value,
  className,
  testId = "popup-work-order-display-key",
}: CopyableKeyButtonProps) {
  const [copied, setCopied] = useState(false);
  const [open, setOpen] = useState(false);
  const timeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    return () => {
      if (timeoutRef.current) clearTimeout(timeoutRef.current);
    };
  }, []);

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(value);
      setCopied(true);
      setOpen(true);
      if (timeoutRef.current) clearTimeout(timeoutRef.current);
      timeoutRef.current = setTimeout(() => {
        setCopied(false);
        setOpen(false);
      }, COPIED_CONFIRMATION_MS);
    } catch {
      showErrorToast("Failed to copy the task ID.");
    }
  };

  return (
    <Tooltip open={open} onOpenChange={setOpen}>
      <TooltipTrigger asChild>
        <span>
          <button
            type="button"
            onClick={() => void handleCopy()}
            className={cn(
              "shrink-0 rounded-sm px-1 py-0.5 font-mono text-[11px] tabular-nums text-muted-foreground hover:bg-muted/60 hover:text-foreground",
              className,
            )}
            aria-label={copied ? "Copied" : `Copy task ID ${value}`}
            data-testid={testId}
          >
            {value}
          </button>
        </span>
      </TooltipTrigger>
      <TooltipContent side="bottom">{copied ? "Copied" : "Copy task ID"}</TooltipContent>
    </Tooltip>
  );
}
