import { Label } from "@/components/ui/label";
import { Link } from "@/components/Link/link";
import { cn } from "@/lib/utils";
import { Switch } from "@/ui/switch";
import { Maximize2 } from "lucide-react";
import { useId } from "react";

import { SectionTitle } from "../work-order-popup-redesign/popupShared";

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
  const followId = useId();

  return (
    <div className={cn("flex items-center justify-between gap-2", className)}>
      <div className="flex min-w-0 items-center gap-3">
        <SectionTitle>Log</SectionTitle>
        <div className="flex items-center gap-1.5">
          <Label htmlFor={followId} className="text-xs font-medium text-muted-foreground">
            Follow
          </Label>
          <Switch id={followId} checked={following} onCheckedChange={onFollowingChange} />
        </div>
      </div>
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
    </div>
  );
}
