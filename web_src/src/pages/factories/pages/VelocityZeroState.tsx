import { TrendingUp } from "lucide-react";
import { Link } from "react-router";

import { Button } from "@/components/ui/button";

const CARD_CLASSES =
  "flex flex-col items-center justify-center gap-3 rounded-lg border border-dashed border-border bg-card px-6 py-16 text-center";

interface VelocityZeroStateProps {
  /** Tasks board for this workspace. Velocity reports work; it does not start it. */
  tasksHref: string;
}

export function VelocityZeroState({ tasksHref }: VelocityZeroStateProps) {
  return (
    <div className={CARD_CLASSES} data-testid="velocity-zero-state">
      <TrendingUp className="size-6 text-muted-foreground" aria-hidden />
      <div className="space-y-1">
        <p className="text-[14px] font-medium text-foreground">No velocity data yet</p>
        <p className="text-[12px] text-muted-foreground">
          Velocity reports merged pull requests, task time, and cost after your first tasks close.
        </p>
      </div>
      <Button asChild variant="outline" size="sm">
        <Link to={tasksHref} data-testid="velocity-zero-state-tasks">
          View tasks
        </Link>
      </Button>
    </div>
  );
}
