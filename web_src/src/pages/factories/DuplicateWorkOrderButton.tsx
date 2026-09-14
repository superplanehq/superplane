import { Copy } from "lucide-react";

import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";

interface DuplicateWorkOrderButtonProps {
  onClick: () => void;
  busy?: boolean;
  className?: string;
  iconClassName?: string;
}

export function DuplicateWorkOrderButton({
  onClick,
  busy = false,
  className,
  iconClassName,
}: DuplicateWorkOrderButtonProps) {
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <span>
          <button
            type="button"
            onClick={onClick}
            disabled={busy}
            className={cn("inline-flex items-center justify-center", className)}
            aria-label="Duplicate"
            data-testid="popup-work-order-duplicate-button"
          >
            <Copy className={iconClassName} aria-hidden />
          </button>
        </span>
      </TooltipTrigger>
      <TooltipContent side="bottom">Duplicate</TooltipContent>
    </Tooltip>
  );
}
