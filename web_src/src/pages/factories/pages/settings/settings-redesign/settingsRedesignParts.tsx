import type { ReactNode } from "react";

import { Avatar } from "@/components/Avatar/avatar";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Label } from "@/components/ui/label";
import { LoadingButton } from "@/components/ui/loading-button";
import { cn } from "@/lib/utils";

export function SettingsIdentityHero({
  initials,
  leading,
  title,
  caption,
  testId = "settings-redesign-identity-hero",
}: {
  initials?: string;
  leading?: ReactNode;
  title: string;
  caption: string;
  testId?: string;
}) {
  return (
    <div className="flex items-center gap-4" data-testid={testId}>
      {leading ?? (
        <Avatar initials={initials || "?"} alt={title} className="size-14 rounded-xl text-[15px] font-medium" />
      )}
      <div className="min-w-0">
        <p className="truncate text-[15px] font-medium tracking-tight text-foreground">{title}</p>
        <p className="truncate font-mono text-[12px] text-muted-foreground">{caption}</p>
      </div>
    </div>
  );
}

export function SettingsStackedField({
  htmlFor,
  label,
  hint,
  children,
}: {
  htmlFor?: string;
  label: string;
  hint?: string;
  children: ReactNode;
}) {
  return (
    <div className="space-y-2">
      <Label htmlFor={htmlFor}>{label}</Label>
      {children}
      {hint ? <p className="text-[12px] text-muted-foreground">{hint}</p> : null}
    </div>
  );
}

export function SettingsUrlField({
  prefix,
  id,
  testId,
  value,
  disabled,
  autoComplete,
  className,
  onChange,
}: {
  prefix: string;
  id: string;
  testId?: string;
  value: string;
  disabled?: boolean;
  autoComplete?: string;
  className?: string;
  onChange: (value: string) => void;
}) {
  return (
    <div className="flex h-9 overflow-hidden rounded-md border border-input bg-background">
      <span className="flex shrink-0 items-center border-r border-input bg-muted px-2.5 font-mono text-[12px] text-muted-foreground">
        {prefix}
      </span>
      <input
        id={id}
        data-testid={testId}
        value={value}
        disabled={disabled}
        autoComplete={autoComplete}
        onChange={(event) => onChange(event.target.value)}
        className={cn(
          "h-full min-w-0 flex-1 bg-transparent px-3 text-[13px] outline-none disabled:opacity-50",
          className,
        )}
      />
    </div>
  );
}

export function SettingsSaveBar({
  allowed,
  permissionsLoading,
  disabled,
  loading,
  onSave,
  denyMessage,
  testId,
}: {
  allowed: boolean;
  permissionsLoading: boolean;
  disabled: boolean;
  loading: boolean;
  onSave: () => void;
  denyMessage: string;
  testId: string;
}) {
  return (
    <div className="flex justify-end border-t border-border pt-4">
      <PermissionTooltip allowed={allowed || permissionsLoading} message={denyMessage}>
        <LoadingButton
          type="button"
          disabled={disabled}
          loading={loading}
          loadingText="Saving..."
          onClick={onSave}
          data-testid={testId}
        >
          Save
        </LoadingButton>
      </PermissionTooltip>
    </div>
  );
}

export function SettingsDangerPanel({
  title,
  description,
  action,
  children,
  testId,
}: {
  title: string;
  description: string;
  action: ReactNode;
  children?: ReactNode;
  testId?: string;
}) {
  return (
    <div className="space-y-3 rounded-lg border border-destructive/40 bg-destructive/5 p-4" data-testid={testId}>
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div className="min-w-0 space-y-1">
          <p className="text-[13px] font-medium text-destructive">{title}</p>
          <p className="text-[12px] text-muted-foreground">{description}</p>
        </div>
        <div className="shrink-0">{action}</div>
      </div>
      {children}
    </div>
  );
}

export function SettingsListRow({
  icon,
  title,
  subtitle,
  meta,
  action,
  testId,
}: {
  icon?: ReactNode;
  title: ReactNode;
  subtitle?: ReactNode;
  meta?: ReactNode;
  action?: ReactNode;
  testId?: string;
}) {
  return (
    <li className="flex items-center gap-3 border-b border-border py-3 last:border-b-0 first:pt-0" data-testid={testId}>
      {icon ? <div className="shrink-0 text-muted-foreground">{icon}</div> : null}
      <div className="min-w-0 flex-1">
        <div className="truncate text-[13px] font-medium text-foreground">{title}</div>
        {subtitle ? <div className="truncate text-[12px] text-muted-foreground">{subtitle}</div> : null}
      </div>
      {meta ? <div className="shrink-0 text-[12px] text-muted-foreground">{meta}</div> : null}
      {action ? <div className="shrink-0">{action}</div> : null}
    </li>
  );
}

export function SettingsStat({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <p className="text-[12px] text-muted-foreground">{label}</p>
      <p className="mt-1 text-[26px] font-medium tracking-tight text-foreground">{value}</p>
    </div>
  );
}
