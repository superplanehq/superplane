import { cn } from "@/lib/utils";
import type { ReactNode } from "react";
import { WorkspacePageHeader } from "../../layout/WorkspacePageHeader";
import { factorySectionBodyClassName, factorySectionHeaderClassName } from "../factoryPageLayoutStyles";

export function FactorySettingsPageFrame({
  title,
  subtitle,
  actions,
  children,
}: {
  title: string;
  subtitle?: ReactNode;
  actions?: ReactNode;
  children: ReactNode;
}) {
  return (
    <>
      <WorkspacePageHeader
        className={factorySectionHeaderClassName}
        title={title}
        subtitle={subtitle}
        actions={actions}
      />
      <div className={cn(factorySectionBodyClassName, "flex flex-col gap-8")}>{children}</div>
    </>
  );
}

export function FactorySettingsCard({
  title,
  titleClassName,
  children,
  className,
  "data-testid": testId,
}: {
  title?: string;
  titleClassName?: string;
  children: ReactNode;
  className?: string;
  "data-testid"?: string;
}) {
  return (
    <section className={cn("rounded-lg border border-border p-4", className)} data-testid={testId}>
      {title ? (
        <h2 className={cn("mb-3 text-[13px] font-medium tracking-[-0.01em] text-foreground", titleClassName)}>
          {title}
        </h2>
      ) : null}
      {children}
    </section>
  );
}
