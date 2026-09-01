import type { ReactNode } from "react";

import { cn } from "@/lib/utils";

export function SettingsActionRow({
  title,
  description,
  action,
  testId,
}: {
  title: string;
  description: ReactNode;
  action: ReactNode;
  testId?: string;
}) {
  return (
    <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between sm:gap-6" data-testid={testId}>
      <div className="min-w-0 space-y-0.5">
        <p className="text-[13px] font-medium text-foreground">{title}</p>
        <div className="text-[12px] text-muted-foreground">{description}</div>
      </div>
      <div className="shrink-0">{action}</div>
    </div>
  );
}

export function SettingsChoice({
  id,
  label,
  description,
  checked,
  onSelect,
}: {
  id: string;
  label: string;
  description: string;
  checked: boolean;
  onSelect: () => void;
}) {
  return (
    <label
      htmlFor={id}
      className={cn(
        "flex cursor-pointer items-start gap-3 rounded-md border px-3 py-2.5",
        checked ? "border-foreground/25 bg-muted/40" : "border-border hover:bg-muted/20",
      )}
    >
      <input id={id} type="radio" checked={checked} onChange={onSelect} className="mt-0.5 size-3.5 accent-foreground" />
      <span>
        <span className="block text-[13px] font-medium text-foreground">{label}</span>
        <span className="mt-0.5 block text-[12px] text-muted-foreground">{description}</span>
      </span>
    </label>
  );
}

export function initialsFor(name: string): string {
  const parts = name
    .split(/\s+/)
    .filter(Boolean)
    .map((part) => part[0]?.toUpperCase() ?? "");
  if (parts.length === 0) {
    return "?";
  }
  if (parts.length === 1) {
    return parts[0];
  }
  return `${parts[0]}${parts[parts.length - 1]}`;
}
