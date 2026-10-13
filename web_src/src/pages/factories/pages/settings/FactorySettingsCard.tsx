import { cn } from "@/lib/utils";
import type { ReactNode } from "react";
import { WorkspacePageHeader } from "../../layout/WorkspacePageHeader";
import {
  factoryCardClassName,
  factorySettingsNarrowSectionBodyClassName,
  factorySettingsNarrowSectionHeaderClassName,
  factorySettingsSectionBodyClassName,
  factorySettingsSectionHeaderClassName,
  factorySettingsWideSectionBodyClassName,
  factorySettingsWideSectionHeaderClassName,
} from "../factoryPageLayoutStyles";

function hasSettingsCardBody(children: ReactNode): boolean {
  return children != null && children !== false;
}

function settingsCardHeaderClassName(attachedList: boolean, hasBody: boolean, hasAction: boolean): string {
  const spacing = attachedList ? "pb-2" : hasBody && "mb-3";
  const alignment = hasAction ? "items-start" : "items-center";
  return cn("flex flex-wrap justify-between gap-3", spacing, alignment);
}

export function FactorySettingsPageFrame({
  title,
  subtitle,
  actions,
  children,
  wide = false,
  narrow = false,
  backHref,
  backLabel,
  backTestId,
}: {
  title: string;
  subtitle?: ReactNode;
  actions?: ReactNode;
  children: ReactNode;
  /** Use the wide column for tables that do not fit the form measure. */
  wide?: boolean;
  /** Shorter column for compact list pages such as Agent. Ignored when `wide` is true. */
  narrow?: boolean;
  /** When set, shows a back link above the title in the page header. */
  backHref?: string;
  backLabel?: string;
  backTestId?: string;
}) {
  const headerClassName = wide
    ? factorySettingsWideSectionHeaderClassName
    : narrow
      ? factorySettingsNarrowSectionHeaderClassName
      : factorySettingsSectionHeaderClassName;
  const bodyClassName = wide
    ? factorySettingsWideSectionBodyClassName
    : narrow
      ? factorySettingsNarrowSectionBodyClassName
      : factorySettingsSectionBodyClassName;
  const hasBack = Boolean(backHref && backLabel);
  return (
    <>
      {hasBack ? (
        <WorkspacePageHeader
          variant="entity"
          className={headerClassName}
          title={title}
          subtitle={subtitle}
          actions={actions}
          backHref={backHref}
          backLabel={backLabel}
          backTestId={backTestId}
        />
      ) : (
        <WorkspacePageHeader className={headerClassName} title={title} subtitle={subtitle} actions={actions} />
      )}
      <div className={cn(bodyClassName, "flex flex-col gap-5")}>{children}</div>
    </>
  );
}

export function FactorySettingsCard({
  title,
  titleClassName,
  description,
  action,
  children,
  className,
  attachedList = false,
  id,
  "data-testid": testId,
}: {
  title?: string;
  titleClassName?: string;
  description?: ReactNode;
  action?: ReactNode;
  children?: ReactNode;
  className?: string;
  /** List rows sit directly under the header with a top border, like integration instances. */
  attachedList?: boolean;
  id?: string;
  "data-testid"?: string;
}) {
  const sectionId = id ?? testId;
  const showHeader = Boolean(title || description || action);
  const hasBody = hasSettingsCardBody(children);
  return (
    <section
      id={sectionId}
      className={cn(factoryCardClassName, "scroll-mt-8 p-4", attachedList && "pb-0", className)}
      data-testid={testId}
    >
      {showHeader ? (
        <div className={settingsCardHeaderClassName(attachedList, hasBody, Boolean(action))}>
          {title || description ? (
            <div className="flex min-w-0 flex-1 flex-col gap-1">
              {title ? (
                <h2 className={cn("text-[13px] font-medium tracking-[-0.01em] text-foreground", titleClassName)}>
                  {title}
                </h2>
              ) : null}
              {description ? <p className="text-[12px] text-muted-foreground">{description}</p> : null}
            </div>
          ) : (
            <div className="min-w-0 flex-1" />
          )}
          {action}
        </div>
      ) : null}
      {hasBody ? children : null}
    </section>
  );
}
