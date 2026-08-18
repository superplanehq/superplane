import type { ReactNode } from "react";
import { Link } from "react-router";

import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";

import { factoryCardClassName } from "../factoryPageLayoutStyles";

/** Shared card scaffold for the Overview redesign sections. */
export function OverviewCard({
  title,
  subtitle,
  count,
  preview,
  headerAction,
  children,
  testId,
}: {
  title: string;
  subtitle?: string;
  count?: number;
  preview?: boolean;
  headerAction?: ReactNode;
  children: ReactNode;
  testId?: string;
}) {
  return (
    <section className={cn("overflow-hidden", factoryCardClassName)} data-testid={testId}>
      <div className="flex items-center justify-between gap-3 border-b border-border px-4 py-3">
        <div className="min-w-0">
          <div className="flex items-center gap-2">
            <h2 className="text-[13px] font-medium tracking-[-0.01em] text-foreground">{title}</h2>
            {count !== undefined && count > 0 ? (
              <span className="rounded-full bg-muted px-1.5 py-0.5 text-[11px] font-medium tabular-nums text-muted-foreground">
                {count}
              </span>
            ) : null}
            {preview ? (
              <Badge variant="outline" className="px-1.5 py-0 text-[10px] font-medium uppercase tracking-[0.05em]">
                Preview
              </Badge>
            ) : null}
          </div>
          {subtitle ? <p className="text-[12px] text-muted-foreground">{subtitle}</p> : null}
        </div>
        {headerAction}
      </div>
      {children}
    </section>
  );
}

export function CardViewAllLink({ href, label }: { href: string; label: string }) {
  return (
    <Link to={href} className="shrink-0 text-[12px] font-medium text-muted-foreground hover:text-foreground">
      {label}
    </Link>
  );
}

export function CardEmptyState({
  icon,
  title,
  hint,
  action,
}: {
  icon?: ReactNode;
  title: string;
  hint?: string;
  action?: ReactNode;
}) {
  return (
    <div className="flex flex-col items-center gap-1.5 px-4 py-8 text-center">
      {icon}
      <p className="text-[13px] font-medium text-foreground">{title}</p>
      {hint ? <p className="max-w-[36ch] text-[12px] text-muted-foreground">{hint}</p> : null}
      {action ? <div className="mt-2">{action}</div> : null}
    </div>
  );
}
