import type { ReactNode } from "react";

import { Switch } from "@/ui/switch";

export function SettingsActionRow({
  title,
  description,
  action,
  testId,
}: {
  title: ReactNode;
  description: ReactNode;
  action: ReactNode;
  testId?: string;
}) {
  return (
    <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between sm:gap-6" data-testid={testId}>
      <div className="min-w-0 space-y-0.5">
        <div className="text-[13px] font-medium text-foreground">{title}</div>
        <div className="text-[12px] text-muted-foreground">{description}</div>
      </div>
      <div className="shrink-0">{action}</div>
    </div>
  );
}

export function SettingsToggleRow({
  title,
  description,
  checked,
  onCheckedChange,
  testId,
}: {
  title: string;
  description: string;
  checked: boolean;
  onCheckedChange: (checked: boolean) => void;
  testId?: string;
}) {
  return (
    <div className="flex items-start justify-between gap-6 py-3" data-testid={testId}>
      <div className="min-w-0 space-y-0.5">
        <p className="text-[13px] font-medium text-foreground">{title}</p>
        <p className="text-[12px] text-muted-foreground">{description}</p>
      </div>
      <Switch checked={checked} onCheckedChange={onCheckedChange} aria-label={title} className="mt-0.5" />
    </div>
  );
}
