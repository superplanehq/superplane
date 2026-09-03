import { Label } from "@/components/ui/label";
import { Link } from "@/components/Link/link";
import { cn } from "@/lib/utils";
import { Switch } from "@/ui/switch";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/ui/tooltip";
import { ChevronsDown, Maximize2 } from "lucide-react";
import { useId } from "react";

import { SectionTitle } from "../work-order-popup-redesign/popupShared";

const FOLLOW_TOOLTIP_TEXT = "Auto-scroll the log to the newest output.";

export function SplitRunFollowSwitch({
  following,
  onFollowingChange,
  className,
}: {
  following: boolean;
  onFollowingChange: (next: boolean) => void;
  className?: string;
}) {
  const followId = useId();
  const followDescriptionId = `${followId}-description`;

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <div className={cn("ml-auto flex items-center gap-1.5", className)} data-testid="split-run-follow">
          <Label htmlFor={followId} className="flex items-center gap-1 text-xs font-medium text-muted-foreground">
            <ChevronsDown className="size-3.5" aria-hidden />
            Follow
          </Label>
          <Switch
            id={followId}
            checked={following}
            onCheckedChange={onFollowingChange}
            aria-describedby={followDescriptionId}
          />
          <span id={followDescriptionId} className="sr-only">
            {FOLLOW_TOOLTIP_TEXT}
          </span>
        </div>
      </TooltipTrigger>
      <TooltipContent side="top">{FOLLOW_TOOLTIP_TEXT}</TooltipContent>
    </Tooltip>
  );
}

export function SplitRunLogHeader({
  following,
  onFollowingChange,
  expandHref,
  className,
}: {
  following: boolean;
  onFollowingChange: (next: boolean) => void;
  expandHref?: string | null;
  className?: string;
}) {
  return (
    <div className={cn("flex items-center justify-between gap-2", className)}>
      <SectionTitle>Automations</SectionTitle>
      {expandHref ? (
        <Link
          href={expandHref}
          aria-label="Open automation run"
          className="flex size-7 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
          data-testid="split-run-log-expand"
        >
          <Maximize2 className="size-3.5" aria-hidden />
        </Link>
      ) : null}
      <SplitRunFollowSwitch following={following} onFollowingChange={onFollowingChange} />
    </div>
  );
}
