import { cn } from "@/lib/utils";
import type { ReactNode } from "react";
import { WorkspacePageHeader } from "../../layout/WorkspacePageHeader";
import {
  factoryCardClassName,
  factorySettingsSectionBodyClassName,
  factorySettingsSectionHeaderClassName,
  factorySettingsWideSectionBodyClassName,
  factorySettingsWideSectionHeaderClassName,
} from "../factoryPageLayoutStyles";

export function FactorySettingsPageFrame({
  title,
  subtitle,
  actions,
  children,
  wide = false,
}: {
  title: string;
  subtitle?: ReactNode;
  actions?: ReactNode;
  children: ReactNode;
  /** Use the wide column for tables that do not fit the form measure. */
  wide?: boolean;
}) {
  const headerClassName = wide ? factorySettingsWideSectionHeaderClassName : factorySettingsSectionHeaderClassName;
  const bodyClassName = wide ? factorySettingsWideSectionBodyClassName : factorySettingsSectionBodyClassName;
  return (
    <>
      <WorkspacePageHeader className={headerClassName} title={title} subtitle={subtitle} actions={actions} />
      <div className={cn(bodyClassName, "flex flex-col gap-5")}>{children}</div>
    </>
  );
}

export function FactorySettingsCard({
  title,
  titleClassName,
  action,
  children,
  className,
  id,
  "data-testid": testId,
}: {
  title?: string;
  titleClassName?: string;
  action?: ReactNode;
  children: ReactNode;
  className?: string;
  id?: string;
  "data-testid"?: string;
}) {
  const sectionId = id ?? testId;
  return (
    <section id={sectionId} className={cn(factoryCardClassName, "scroll-mt-8 p-4", className)} data-testid={testId}>
      {title || action ? (
        <div className="mb-3 flex items-center justify-between gap-3">
          {title ? (
            <h2 className={cn("text-[13px] font-medium tracking-[-0.01em] text-foreground", titleClassName)}>
              {title}
            </h2>
          ) : null}
          {action}
        </div>
      ) : null}
      {children}
    </section>
  );
}
