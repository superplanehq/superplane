import type { FactoriesFactoryIntakeItem } from "@/api-client";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Label } from "@/components/ui/label";
import { useSearchDependabotIntakeSetupItems } from "@/hooks/useFactoryIntakeData";
import { getApiErrorMessage } from "@/lib/errors";
import { githubRepositorySecurityAnalysisUrl } from "@/lib/githubRepository";
import { cn } from "@/lib/utils";
import { ExternalLink } from "lucide-react";
import { useId, useState } from "react";

import { DEPENDABOT_INTAKE_SETUP_COPY } from "./dependabotIntakeSetupCopy";
import { FirstRunHeading } from "./onboarding/first-run/FirstRunShell";

const COPY = DEPENDABOT_INTAKE_SETUP_COPY.import;

/** The intake source clamps the page size to 50. One repository rarely has more packages than this. */
const PACKAGE_LIMIT = 50;

interface DependabotIntakeImportStepProps {
  organizationId: string;
  factoryId: string;
  repository: string;
  dependabotSeverities: string[];
  importProgress?: { done: number; total: number };
  error?: string;
  onSkip: () => void;
  onChangeFilters: () => void;
  onImportSelected: (packageIds: string[]) => void;
}

/**
 * Second Dependabot setup step. Lists packages with open alerts. The parent
 * creates the intake when the user skips or imports.
 */
export function DependabotIntakeImportStep(props: DependabotIntakeImportStepProps) {
  const items = useSearchDependabotIntakeSetupItems({
    organizationId: props.organizationId,
    factoryId: props.factoryId,
    dependabotSeverities: props.dependabotSeverities,
    limit: PACKAGE_LIMIT,
  });
  const [selected, setSelected] = useState<Set<string>>(new Set());

  const packages = (items.data ?? []).filter((item): item is FactoriesFactoryIntakeItem & { id: string } =>
    Boolean(item.id),
  );
  const finishing = props.importProgress !== undefined;
  const listReady = !items.isLoading && !items.isError;
  const emptyList = listReady && packages.length === 0;
  const allSelected = packages.length > 0 && packages.every((item) => selected.has(item.id));

  const toggle = (id: string) => {
    setSelected((current) => {
      const next = new Set(current);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  };

  const toggleAll = () => {
    setSelected(allSelected ? new Set() : new Set(packages.map((item) => item.id)));
  };

  const importSelected = () => {
    const ids = packages.map((item) => item.id).filter((id) => selected.has(id));
    if (ids.length === 0) return;
    props.onImportSelected(ids);
  };

  return (
    <>
      <FirstRunHeading headline={COPY.pageTitle}>
        <p className="text-[15px] leading-6 text-muted-foreground">{COPY.helper}</p>
      </FirstRunHeading>

      <div className="mt-8 space-y-4">
        <div className="rounded-xl border border-border bg-card px-4 py-4 text-left">
          <PackageListHeader
            items={items}
            repository={props.repository}
            packageCount={packages.length}
            allSelected={allSelected}
            disabled={finishing}
            onToggleAll={toggleAll}
          />
          {packages.length > 0 ? (
            <ul className="mt-3 space-y-2" data-testid="dependabot-import-packages">
              {packages.map((item) => (
                <PackageRow
                  key={item.id}
                  item={item}
                  checked={selected.has(item.id)}
                  disabled={finishing}
                  onToggle={() => toggle(item.id)}
                />
              ))}
            </ul>
          ) : null}
        </div>

        {props.error ? (
          <p className="workspace-body-text text-destructive" role="alert">
            {props.error}
          </p>
        ) : null}

        {emptyList ? (
          <>
            <Button
              type="button"
              className="w-full"
              disabled={finishing}
              onClick={props.onSkip}
              data-testid="dependabot-import-finish-empty"
            >
              {COPY.finishWithoutImporting}
            </Button>
            <Button
              type="button"
              variant="ghost"
              className="w-full"
              disabled={finishing}
              onClick={props.onChangeFilters}
              data-testid="dependabot-import-change-filters"
            >
              {COPY.changeFilters}
            </Button>
          </>
        ) : (
          <>
            <Button
              type="button"
              className="w-full"
              disabled={finishing || selected.size === 0}
              onClick={importSelected}
              data-testid="dependabot-import-selected"
            >
              {props.importProgress
                ? COPY.importing(props.importProgress.done + 1, props.importProgress.total)
                : COPY.importSelected(selected.size)}
            </Button>
            <Button
              type="button"
              variant="ghost"
              className="w-full"
              disabled={finishing}
              onClick={props.onSkip}
              data-testid="dependabot-import-skip"
            >
              {COPY.skip}
            </Button>
          </>
        )}
      </div>
    </>
  );
}

function PackageListHeader({
  items,
  repository,
  packageCount,
  allSelected,
  disabled,
  onToggleAll,
}: {
  items: ReturnType<typeof useSearchDependabotIntakeSetupItems>;
  repository: string;
  packageCount: number;
  allSelected: boolean;
  disabled: boolean;
  onToggleAll: () => void;
}) {
  if (items.isLoading) {
    return (
      <p className="text-[13px] text-muted-foreground" data-testid="dependabot-import-loading">
        {COPY.loading}
      </p>
    );
  }
  if (items.isError) {
    const message = getApiErrorMessage(items.error, COPY.loadError);
    const securitySettingsUrl =
      message === COPY.alertsDisabledMessage ? githubRepositorySecurityAnalysisUrl(repository) : undefined;

    return (
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0 space-y-2">
          <p className="text-[13px] text-destructive" role="alert">
            {message}
          </p>
          {securitySettingsUrl ? (
            <a
              href={securitySettingsUrl}
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex items-center gap-1.5 text-[13px] font-medium text-foreground underline-offset-4 hover:underline"
              data-testid="dependabot-open-security-settings"
            >
              {COPY.openSecuritySettings}
              <ExternalLink className="size-3.5 shrink-0" aria-hidden />
            </a>
          ) : null}
        </div>
        <Button type="button" variant="outline" size="sm" onClick={() => void items.refetch()}>
          {COPY.retry}
        </Button>
      </div>
    );
  }
  if (packageCount === 0) {
    return (
      <div className="space-y-1" data-testid="dependabot-import-empty">
        <p className="text-[13px] font-medium text-foreground">{COPY.emptyTitle}</p>
        <p className="text-[13px] text-muted-foreground">{COPY.emptyBody}</p>
      </div>
    );
  }
  return (
    <div className="flex items-center justify-between gap-3">
      <p className="text-[13px] font-medium" data-testid="dependabot-import-count">
        {COPY.matchCount(packageCount)}
      </p>
      <Button
        type="button"
        variant="ghost"
        size="sm"
        disabled={disabled}
        onClick={onToggleAll}
        data-testid="dependabot-import-select-all"
      >
        {allSelected ? COPY.clearSelection : COPY.selectAll}
      </Button>
    </div>
  );
}

function PackageRow({
  item,
  checked,
  disabled,
  onToggle,
}: {
  item: FactoriesFactoryIntakeItem & { id: string };
  checked: boolean;
  disabled: boolean;
  onToggle: () => void;
}) {
  const id = useId();
  return (
    <li>
      <Label
        htmlFor={id}
        className={cn(
          "flex cursor-pointer items-center gap-3 rounded-lg border px-3 py-2.5 transition-colors",
          checked ? "border-foreground/20 bg-accent/50" : "border-border bg-card hover:border-foreground/15",
          disabled && "cursor-default",
        )}
      >
        <Checkbox id={id} checked={checked} disabled={disabled} onChange={onToggle} />
        <span className="min-w-0 flex-1 truncate text-[13px] font-medium tracking-[-0.01em] text-foreground">
          {item.title}
        </span>
        {item.key ? <span className="shrink-0 text-[12px] text-muted-foreground">{item.key}</span> : null}
      </Label>
    </li>
  );
}
