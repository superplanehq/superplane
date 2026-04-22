import { useEffect, useRef, useState } from "react";
import { Check, Copy } from "lucide-react";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

const RESET_DELAY_MS = 2000;

interface CopyButtonProps {
  text: string;
  /** "icon" (default) renders a compact icon-only button;
   *  "button" renders a labeled outline button for primary copy actions. */
  variant?: "icon" | "button";
  children?: React.ReactNode;
  /** Label briefly shown after a successful copy (button variant). */
  copiedLabel?: React.ReactNode;
  /** Inverts icon colors on dark backgrounds (icon variant only). */
  dark?: boolean;
  /** Fires when `navigator.clipboard.writeText` rejects. */
  onCopyError?: (err: unknown) => void;
  className?: string;
  "data-testid"?: string;
}

export function CopyButton({
  text,
  variant = "icon",
  children,
  copiedLabel = "Copied!",
  dark,
  onCopyError,
  className,
  "data-testid": dataTestId,
}: CopyButtonProps) {
  const [copied, setCopied] = useState(false);
  const timeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    return () => {
      if (timeoutRef.current) clearTimeout(timeoutRef.current);
    };
  }, []);

  const handleCopy = async (e: React.MouseEvent) => {
    e.stopPropagation();
    try {
      await navigator.clipboard.writeText(text);
    } catch (err) {
      onCopyError?.(err);
      return;
    }
    setCopied(true);
    if (timeoutRef.current) clearTimeout(timeoutRef.current);
    timeoutRef.current = setTimeout(() => setCopied(false), RESET_DELAY_MS);
  };

  if (variant === "button") {
    return (
      <Button
        type="button"
        variant="outline"
        onClick={handleCopy}
        aria-live="polite"
        data-testid={dataTestId}
        className={cn("flex items-center gap-1", className)}
      >
        {copied ? (
          <>
            <Check className="text-green-600 dark:text-green-400" />
            {copiedLabel}
          </>
        ) : (
          <>
            <Copy />
            {children ?? "Copy"}
          </>
        )}
      </Button>
    );
  }

  return (
    <button
      type="button"
      onClick={handleCopy}
      aria-label={copied ? "Copied to clipboard" : "Copy to clipboard"}
      aria-live="polite"
      data-testid={dataTestId}
      className={cn(
        "p-1 rounded transition-colors shrink-0",
        dark ? "hover:bg-gray-700" : "hover:bg-gray-200 dark:hover:bg-gray-700",
        className,
      )}
      title="Copy to clipboard"
    >
      {copied ? (
        <Check size={13} className={dark ? "text-green-400" : "text-green-600 dark:text-green-400"} />
      ) : (
        <Copy size={13} className={dark ? "text-gray-400 hover:text-gray-200" : "text-gray-400"} />
      )}
    </button>
  );
}
