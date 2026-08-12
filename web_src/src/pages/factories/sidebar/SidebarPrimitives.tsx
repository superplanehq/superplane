import type { ReactNode } from "react";

export function SidebarSectionHeading({ children }: { children: ReactNode }) {
  return <h3 className="workspace-section-label">{children}</h3>;
}

export function OverviewRow({
  icon,
  srLabel,
  children,
}: {
  icon: ReactNode;
  srLabel: string;
  children: ReactNode;
}) {
  return (
    <div className="flex items-center gap-2 py-1.5">
      <span className="shrink-0 text-muted-foreground" aria-hidden>
        {icon}
      </span>
      <div className="min-w-0 flex-1 text-[13px] tracking-[-0.01em] text-foreground">
        <span className="sr-only">{srLabel}</span>
        {children}
      </div>
    </div>
  );
}
