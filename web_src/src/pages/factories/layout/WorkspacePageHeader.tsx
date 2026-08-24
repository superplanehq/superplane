import type { ReactNode } from "react";
import { ArrowLeft } from "lucide-react";
import { Link } from "@/components/Link/link";
import { cn } from "@/lib/utils";
import { factoryContentHeaderClassName } from "../pages/factoryPageLayoutStyles";

interface WorkspacePageHeaderBaseProps {
  /** Optional wrapper className appended to the shared shell. */
  className?: string;
  /** Optional test id on the outer header element. */
  "data-testid"?: string;
  /** Trailing actions. Default is right-aligned; `start` keeps them in the title row. */
  actions?: ReactNode;
  /** Where to place `actions` in the title row. */
  actionsAlign?: "start" | "end";
  /** Optional content stacked below the title/actions row (filter chips, tabs, etc.). */
  belowRow?: ReactNode;
}

interface WorkspaceSectionHeaderProps extends WorkspacePageHeaderBaseProps {
  variant?: "section";
  /** Section title, e.g. "Work Orders", "Lines". */
  title: ReactNode;
  /** Optional description below the title. */
  subtitle?: ReactNode;
  /** Optional inline element rendered next to the title (e.g. Work Orders scope pills). */
  leading?: ReactNode;
}

interface WorkspaceEntityHeaderProps extends WorkspacePageHeaderBaseProps {
  variant: "entity";
  /** Entity title, e.g. work order title, line name, automation name. */
  title: ReactNode;
  /** Optional short identifier shown above the title (e.g. "SP-42"). */
  kicker?: ReactNode;
  /** Optional description below the title. */
  subtitle?: ReactNode;
  /** Back link target. Omit on landing surfaces that have no parent list. */
  backHref?: string;
  /** Label for the back link, e.g. "Work Orders", "Lines", "Automations". */
  backLabel?: string;
  /** Optional test id on the back link. */
  backTestId?: string;
}

export type WorkspacePageHeaderProps = WorkspaceSectionHeaderProps | WorkspaceEntityHeaderProps;

/**
 * Shared page-chrome header for every top-level `/workspaces/:factoryKey`
 * route. Two variants:
 *
 * - `section` (default): title + optional subtitle + trailing actions. Used
 *   for Overview, Work Orders list, Lines list, Automations list, Wiki,
 *   Velocity, and other section pages.
 * - `entity`: back link + optional identifier kicker + entity title +
 *   trailing actions. Used for work order, line, and automation detail.
 *
 * The header always uses the same gutter, max width, and vertical rhythm
 * so pages match. Callers put body content in `factoryContentBodyClassName`
 * after this header.
 */
export function WorkspacePageHeader(props: WorkspacePageHeaderProps) {
  const isEntity = props.variant === "entity";
  const hasSubtitle = Boolean(props.subtitle);
  const actionsAlign = props.actionsAlign ?? "end";
  const alignItems = hasSubtitle || isEntity ? "items-start" : "items-center";
  return (
    <header
      className={cn(factoryContentHeaderClassName, props.className)}
      data-testid={props["data-testid"] ?? "workspace-page-header"}
    >
      <div
        className={cn(
          "flex w-full flex-wrap gap-x-4 gap-y-2",
          actionsAlign === "start" ? "justify-start" : "justify-between",
          alignItems,
        )}
      >
        <div className={cn("min-w-0", actionsAlign === "start" ? "shrink-0" : "flex-1")}>
          {isEntity && props.backHref && props.backLabel ? (
            <BackLink href={props.backHref} label={props.backLabel} testId={props.backTestId} />
          ) : null}
          {isEntity && props.kicker ? (
            <p
              className="mt-2 text-[12px] font-medium uppercase tracking-[0.06em] text-muted-foreground tabular-nums"
              data-testid="workspace-page-header-kicker"
            >
              {props.kicker}
            </p>
          ) : null}
          <div className={cn("flex min-w-0 flex-wrap items-center gap-3", isEntity && "mt-1")}>
            <h1 className="workspace-page-title min-w-0 overflow-visible" data-testid="workspace-page-header-title">
              {props.title}
            </h1>
            {!isEntity && props.leading ? <div className="flex items-center gap-2">{props.leading}</div> : null}
          </div>
          {hasSubtitle ? (
            <p className="workspace-body-text mt-1 text-muted-foreground" data-testid="workspace-page-header-subtitle">
              {props.subtitle}
            </p>
          ) : null}
        </div>
        {props.actions ? (
          <div className="flex shrink-0 items-center gap-2" data-testid="workspace-page-header-actions">
            {props.actions}
          </div>
        ) : null}
      </div>
      {props.belowRow}
    </header>
  );
}

function BackLink({ href, label, testId }: { href: string; label: string; testId?: string }) {
  return (
    <Link
      href={href}
      className="inline-flex items-center gap-1.5 text-[13px] text-muted-foreground transition-colors hover:text-foreground"
      data-testid={testId ?? "workspace-page-header-back"}
    >
      <ArrowLeft className="size-3.5" strokeWidth={1.75} aria-hidden />
      {label}
    </Link>
  );
}
