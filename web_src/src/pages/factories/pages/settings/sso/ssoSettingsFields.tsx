import { useId, type ReactNode } from "react";

import { PermissionTooltip } from "@/components/PermissionGate";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { LoadingButton } from "@/components/ui/loading-button";
import { Switch } from "@/components/ui/switch";
import { cn } from "@/lib/utils";
import { CopyButton } from "@/ui/CopyButton";

const permissionMessage = "You do not have permission to update Single Sign-On.";

export function TextField({
  id,
  label,
  value,
  onChange,
  disabled,
  helper,
  error,
  type = "text",
  placeholder,
  className = "w-full",
  wrapperClassName,
}: {
  id: string;
  label: string;
  value: string;
  onChange: (value: string) => void;
  disabled?: boolean;
  helper?: string;
  error?: string | null;
  type?: string;
  placeholder?: string;
  className?: string;
  wrapperClassName?: string;
}) {
  return (
    <div className={cn("space-y-2", wrapperClassName)}>
      <Label htmlFor={id}>{label}</Label>
      <Input
        id={id}
        type={type}
        value={value}
        placeholder={placeholder}
        disabled={disabled}
        onChange={(event) => onChange(event.target.value)}
        className={className}
      />
      {helper ? <p className="text-[12px] text-muted-foreground">{helper}</p> : null}
      {error ? <p className="text-[11px] text-destructive">{error}</p> : null}
    </div>
  );
}

export function CopyField({ label, value, helper }: { label: string; value: string; helper?: string }) {
  const id = useId();
  return (
    <div className="space-y-2">
      <Label htmlFor={id}>{label}</Label>
      {helper ? <p className="text-[12px] text-muted-foreground">{helper}</p> : null}
      <div className="flex w-full items-center gap-2">
        <Input id={id} readOnly value={value} className="w-auto min-w-0 flex-1 font-mono text-[12px]" />
        <CopyButton variant="icon" text={value} ariaLabel={`Copy ${label}`} />
      </div>
    </div>
  );
}

export function SwitchRow({
  id,
  title,
  description,
  checked,
  disabled,
  onCheckedChange,
}: {
  id: string;
  title: string;
  description: string;
  checked: boolean;
  disabled?: boolean;
  onCheckedChange: (checked: boolean) => void;
}) {
  return (
    <div className="flex items-start justify-between gap-6">
      <div className="min-w-0 flex-1 space-y-0.5">
        <p className="text-[13px] font-medium text-foreground">{title}</p>
        <p className="text-[12px] leading-5 text-muted-foreground">{description}</p>
      </div>
      <Switch
        id={id}
        checked={checked}
        disabled={disabled}
        onCheckedChange={onCheckedChange}
        className="mt-0.5 shrink-0"
      />
    </div>
  );
}

export function ActionRow({
  title,
  description,
  detail,
  children,
}: {
  title: string;
  description: string;
  detail?: ReactNode;
  children: ReactNode;
}) {
  return (
    <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
      <div className="min-w-0 flex-1 space-y-0.5">
        <p className="text-[13px] font-medium text-foreground">{title}</p>
        <p className="text-[12px] leading-5 text-muted-foreground">{description}</p>
        {detail}
      </div>
      <div className="shrink-0">{children}</div>
    </div>
  );
}

export function CardSave({
  permitted,
  disabled,
  busy,
  onClick,
  label = "Save",
  testId,
}: {
  permitted: boolean;
  disabled: boolean;
  busy: boolean;
  onClick: () => void;
  label?: string;
  testId?: string;
}) {
  return (
    <PermissionTooltip allowed={permitted} message={permissionMessage}>
      <LoadingButton
        type="button"
        disabled={disabled}
        loading={busy}
        loadingText="Saving..."
        onClick={onClick}
        data-testid={testId}
      >
        {label}
      </LoadingButton>
    </PermissionTooltip>
  );
}

export function NeedsConnection() {
  return <p className="text-[12px] text-muted-foreground">Create the connection first.</p>;
}

export function PermissionTip({ permitted, children }: { permitted: boolean; children: ReactNode }) {
  return (
    <PermissionTooltip allowed={permitted} message={permissionMessage}>
      {children}
    </PermissionTooltip>
  );
}
