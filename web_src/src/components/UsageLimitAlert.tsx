import { TriangleAlert } from "lucide-react";
import { Link } from "react-router-dom";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import type { UsageLimitNotice } from "@/utils/usageLimits";
import { Alert, AlertDescription, AlertTitle } from "@/ui/alert";

interface UsageLimitAlertProps {
  notice: UsageLimitNotice;
  className?: string;
}

export function UsageLimitAlert({ notice, className }: UsageLimitAlertProps) {
  return (
    <Alert
      className={cn(
        "border-amber-200 bg-amber-50/90 text-amber-950 [&>svg]:text-amber-700 dark:border-amber-700/50 dark:bg-amber-950/30 dark:text-amber-100 dark:[&>svg]:text-amber-300",
        className,
      )}
    >
      <TriangleAlert className="h-4 w-4" />
      <AlertTitle>{notice.title}</AlertTitle>
      <AlertDescription className="space-y-3">
        <p>{notice.description}</p>
        {notice.href ? (
          <div>
            <Button
              asChild
              variant="outline"
              size="sm"
              className="border-amber-300 bg-amber-100/70 text-amber-950 hover:bg-amber-100 dark:border-amber-600 dark:bg-amber-900/40 dark:text-amber-50 dark:hover:bg-amber-900/60"
            >
              <Link to={notice.href}>{notice.actionLabel || "View usage"}</Link>
            </Button>
          </div>
        ) : null}
      </AlertDescription>
    </Alert>
  );
}
