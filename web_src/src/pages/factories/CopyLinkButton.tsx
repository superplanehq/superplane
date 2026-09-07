import { Check, Link2 } from "lucide-react";
import { useEffect, useRef, useState } from "react";

import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { showErrorToast } from "@/lib/toast";
import { cn } from "@/lib/utils";

/** How long the "Copied" confirmation stays visible. */
const COPIED_CONFIRMATION_MS = 1600;

interface CopyLinkButtonProps {
  /** Link to copy. Defaults to the current page URL. */
  url?: string;
  /** Classes for the trigger button. */
  className?: string;
  /** Classes for the Link2/Check icon. */
  iconClassName?: string;
  ariaLabel?: string;
  testId?: string;
}

/**
 * Icon button that copies a link to the clipboard. Confirms with a tooltip
 * that reads "Copied" and swaps the icon to a check mark for about 1.6 s.
 *
 * Radix Tooltip closes on click by default, so `open` is controlled here to
 * keep the confirmation visible for the full window.
 */
export function CopyLinkButton({
  url,
  className,
  iconClassName,
  ariaLabel = "Copy link to task",
  testId = "work-order-copy-link-button",
}: CopyLinkButtonProps) {
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
      await navigator.clipboard.writeText(url ?? window.location.href);
      setCopied(true);
      setOpen(true);
      if (timeoutRef.current) clearTimeout(timeoutRef.current);
      timeoutRef.current = setTimeout(() => {
        setCopied(false);
        setOpen(false);
      }, COPIED_CONFIRMATION_MS);
    } catch {
      showErrorToast("Failed to copy link.");
    }
  };

  return (
    <Tooltip open={open} onOpenChange={setOpen}>
      <TooltipTrigger asChild>
        <span>
          <button
            type="button"
            onClick={() => void handleCopy()}
            className={cn("inline-flex items-center justify-center", className)}
            aria-label={ariaLabel}
            data-testid={testId}
          >
            {copied ? <Check className={iconClassName} aria-hidden /> : <Link2 className={iconClassName} aria-hidden />}
          </button>
        </span>
      </TooltipTrigger>
      <TooltipContent side="bottom">{copied ? "Copied" : "Copy link"}</TooltipContent>
    </Tooltip>
  );
}
