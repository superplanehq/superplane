import type { FactoriesFactory } from "@/api-client";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast } from "@/lib/toast";
import { useEffect, useState } from "react";

import { VELOCITY_PERIOD_DAYS, type VelocityPeriodDays } from "../../lib/factoryVelocityReport";
import { FactorySettingsCard } from "./FactorySettingsCard";

const PUBLIC_BADGE_COPY = {
  title: "Public badge",
  enabledLabel: "Public badge",
  enabledHelper: "Anyone with the link can see the share, trend, and merge rate.",
  costLabel: "Show cost per merged PR",
  costHelper: "Anyone with the link can see the cost per merged pull request on the Large and Full width badges.",
  periodLabel: "Time frame",
  sizeLabel: "Size",
  markdownLabel: "Markdown",
  markdownHelper: "Paste this snippet into a README.",
  copy: "Copy Markdown",
  previewAlt: "Public badge preview",
  saveError: "SuperPlane could not update the public badge.",
  permission: "You don't have permission to update workspaces.",
} as const;

const BADGE_SIZES = [
  { value: "small", label: "Small" },
  { value: "large", label: "Large" },
  { value: "wide", label: "Full width" },
] as const;

type BadgeSize = (typeof BADGE_SIZES)[number]["value"];

type PublicBadgeUpdate = {
  publicBadgeEnabled?: boolean;
  publicBadgeShowCost?: boolean;
};

export function PublicBadgeSettingsSection({
  factory,
  canUpdate,
  permissionsLoading,
  isSaving,
  onUpdate,
}: {
  factory: FactoriesFactory;
  canUpdate: boolean;
  permissionsLoading: boolean;
  isSaving: boolean;
  onUpdate: (input: PublicBadgeUpdate) => Promise<FactoriesFactory>;
}) {
  const badge = usePublicBadgeState(factory, onUpdate);
  const locked = !canUpdate || isSaving;

  return (
    <FactorySettingsCard
      title={PUBLIC_BADGE_COPY.title}
      id="factory-settings-public-badge"
      data-testid="factory-settings-public-badge"
      className="scroll-mt-8"
    >
      <div className="space-y-4">
        <BadgeSwitchRow
          label={PUBLIC_BADGE_COPY.enabledLabel}
          helper={PUBLIC_BADGE_COPY.enabledHelper}
          checked={badge.enabled}
          disabled={locked}
          canUpdate={canUpdate}
          permissionsLoading={permissionsLoading}
          testId="factory-settings-public-badge-toggle"
          onCheckedChange={(next) => badge.saveEnabled(next)}
        />
        {badge.enabled ? (
          <PublicBadgeDetails
            badge={badge}
            locked={locked}
            canUpdate={canUpdate}
            permissionsLoading={permissionsLoading}
          />
        ) : null}
      </div>
    </FactorySettingsCard>
  );
}

function usePublicBadgeState(
  factory: FactoriesFactory,
  onUpdate: (input: PublicBadgeUpdate) => Promise<FactoriesFactory>,
) {
  const [enabled, setEnabled] = useState(Boolean(factory.publicBadgeEnabled));
  const [showCost, setShowCost] = useState(Boolean(factory.publicBadgeShowCost));
  const [token, setToken] = useState(factory.publicBadgeToken ?? "");
  const [period, setPeriod] = useState<VelocityPeriodDays>(30);
  const [size, setSize] = useState<BadgeSize>("small");
  const [previewNonce, setPreviewNonce] = useState(0);

  useEffect(() => {
    setEnabled(Boolean(factory.publicBadgeEnabled));
    setShowCost(Boolean(factory.publicBadgeShowCost));
    setToken(factory.publicBadgeToken ?? "");
  }, [factory.publicBadgeEnabled, factory.publicBadgeShowCost, factory.publicBadgeToken]);

  const origin = window.location.origin;
  const imageURL = token ? `${origin}/api/v1/public/badges/${token}.svg?period=${period}&size=${size}` : "";

  const save = async (input: PublicBadgeUpdate, apply: () => void, revert: () => void) => {
    apply();
    try {
      const saved = await onUpdate(input);
      if (saved.publicBadgeToken) {
        setToken(saved.publicBadgeToken);
      }
      setPreviewNonce((current) => current + 1);
    } catch (error) {
      revert();
      showErrorToast(getApiErrorMessage(error, PUBLIC_BADGE_COPY.saveError));
    }
  };

  return {
    enabled,
    showCost,
    period,
    size,
    markdown: imageURL ? `[![PRs via SuperPlane](${imageURL})](${origin})` : "",
    previewURL: imageURL ? `${imageURL}&v=${previewNonce}` : "",
    setPeriod,
    setSize,
    saveEnabled: (next: boolean) => {
      void save(
        { publicBadgeEnabled: next },
        () => setEnabled(next),
        () => setEnabled(!next),
      );
    },
    saveShowCost: (next: boolean) => {
      void save(
        { publicBadgeShowCost: next },
        () => setShowCost(next),
        () => setShowCost(!next),
      );
    },
  };
}

type BadgeState = ReturnType<typeof usePublicBadgeState>;

function PublicBadgeDetails({
  badge,
  locked,
  canUpdate,
  permissionsLoading,
}: {
  badge: BadgeState;
  locked: boolean;
  canUpdate: boolean;
  permissionsLoading: boolean;
}) {
  return (
    <div className="space-y-4 border-t border-border pt-4">
      <BadgeSwitchRow
        label={PUBLIC_BADGE_COPY.costLabel}
        helper={PUBLIC_BADGE_COPY.costHelper}
        checked={badge.showCost}
        disabled={locked}
        canUpdate={canUpdate}
        permissionsLoading={permissionsLoading}
        testId="factory-settings-public-badge-cost"
        onCheckedChange={badge.saveShowCost}
      />
      <BadgeSelectors
        period={badge.period}
        size={badge.size}
        locked={locked}
        onPeriodChange={badge.setPeriod}
        onSizeChange={badge.setSize}
      />
      <BadgePreview previewURL={badge.previewURL} />
      <BadgeMarkdown
        markdown={badge.markdown}
        locked={locked}
        canUpdate={canUpdate}
        permissionsLoading={permissionsLoading}
      />
    </div>
  );
}

function BadgeSelectors({
  period,
  size,
  locked,
  onPeriodChange,
  onSizeChange,
}: {
  period: VelocityPeriodDays;
  size: BadgeSize;
  locked: boolean;
  onPeriodChange: (period: VelocityPeriodDays) => void;
  onSizeChange: (size: BadgeSize) => void;
}) {
  return (
    <div className="grid gap-4 sm:grid-cols-2">
      <div className="space-y-2">
        <Label htmlFor="factory-settings-public-badge-period">{PUBLIC_BADGE_COPY.periodLabel}</Label>
        <Select
          value={String(period)}
          disabled={locked}
          onValueChange={(value) => {
            const next = Number(value);
            if (VELOCITY_PERIOD_DAYS.some((days) => days === next)) {
              onPeriodChange(next as VelocityPeriodDays);
            }
          }}
        >
          <SelectTrigger id="factory-settings-public-badge-period" data-testid="factory-settings-public-badge-period">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {VELOCITY_PERIOD_DAYS.map((days) => (
              <SelectItem key={days} value={String(days)}>
                {days} days
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
      <div className="space-y-2">
        <Label htmlFor="factory-settings-public-badge-size">{PUBLIC_BADGE_COPY.sizeLabel}</Label>
        <Select value={size} disabled={locked} onValueChange={(value) => onSizeChange(value as BadgeSize)}>
          <SelectTrigger id="factory-settings-public-badge-size" data-testid="factory-settings-public-badge-size">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {BADGE_SIZES.map((option) => (
              <SelectItem key={option.value} value={option.value}>
                {option.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
    </div>
  );
}

function BadgePreview({ previewURL }: { previewURL: string }) {
  if (!previewURL) {
    return null;
  }
  return (
    <div className="overflow-hidden rounded-md border border-border bg-muted/40 p-4">
      <img
        src={previewURL}
        alt={PUBLIC_BADGE_COPY.previewAlt}
        data-testid="factory-settings-public-badge-preview"
        className="max-w-full"
      />
    </div>
  );
}

function BadgeMarkdown({
  markdown,
  locked,
  canUpdate,
  permissionsLoading,
}: {
  markdown: string;
  locked: boolean;
  canUpdate: boolean;
  permissionsLoading: boolean;
}) {
  if (!markdown) {
    return null;
  }
  return (
    <div className="space-y-2">
      <Label htmlFor="factory-settings-public-badge-markdown">{PUBLIC_BADGE_COPY.markdownLabel}</Label>
      <div className="flex flex-col gap-2 sm:flex-row">
        <Input
          id="factory-settings-public-badge-markdown"
          data-testid="factory-settings-public-badge-markdown"
          value={markdown}
          readOnly
          disabled={locked}
        />
        <PermissionTooltip allowed={canUpdate || permissionsLoading} message={PUBLIC_BADGE_COPY.permission}>
          <Button
            type="button"
            variant="outline"
            disabled={locked}
            data-testid="factory-settings-public-badge-copy"
            onClick={() => void navigator.clipboard.writeText(markdown)}
          >
            {PUBLIC_BADGE_COPY.copy}
          </Button>
        </PermissionTooltip>
      </div>
      <p className="text-[12px] text-muted-foreground">{PUBLIC_BADGE_COPY.markdownHelper}</p>
    </div>
  );
}

function BadgeSwitchRow({
  label,
  helper,
  checked,
  disabled,
  canUpdate,
  permissionsLoading,
  testId,
  onCheckedChange,
}: {
  label: string;
  helper: string;
  checked: boolean;
  disabled: boolean;
  canUpdate: boolean;
  permissionsLoading: boolean;
  testId: string;
  onCheckedChange: (checked: boolean) => void;
}) {
  return (
    <div className="flex items-start justify-between gap-6" data-testid={testId}>
      <div className="min-w-0 space-y-0.5">
        <p className="text-[13px] font-medium text-foreground">{label}</p>
        <p className="text-[12px] leading-5 text-muted-foreground">{helper}</p>
      </div>
      <PermissionTooltip allowed={canUpdate || permissionsLoading} message={PUBLIC_BADGE_COPY.permission}>
        <Switch
          checked={checked}
          disabled={disabled}
          onCheckedChange={onCheckedChange}
          aria-label={label}
          className="mt-0.5"
        />
      </PermissionTooltip>
    </div>
  );
}
