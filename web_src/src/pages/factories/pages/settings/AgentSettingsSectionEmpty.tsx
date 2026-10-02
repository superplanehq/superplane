import type { LucideIcon } from "lucide-react";

/** Compact empty state inside an Agent settings card (action lives in the card header). */
export function AgentSettingsSectionEmpty({
  icon: Icon,
  title,
  description,
  testId,
}: {
  icon: LucideIcon;
  title: string;
  description: string;
  testId?: string;
}) {
  return (
    <div className="py-6 text-center" data-testid={testId}>
      <div className="mx-auto mb-2.5 flex size-8 items-center justify-center rounded-md bg-muted/50 text-muted-foreground">
        <Icon className="size-3.5" strokeWidth={1.75} aria-hidden />
      </div>
      <p className="text-[13px] font-medium text-foreground">{title}</p>
      <p className="mx-auto mt-1 max-w-xs text-[12px] leading-relaxed text-muted-foreground">{description}</p>
    </div>
  );
}
