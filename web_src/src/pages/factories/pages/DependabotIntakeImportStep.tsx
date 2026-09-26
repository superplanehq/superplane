import type { FactoriesFactoryIntakeItem } from "@/api-client";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Label } from "@/components/ui/label";
import { useImportFactoryIntakeItem, useSearchFactoryIntakeItems } from "@/hooks/useFactoryIntakeData";
import { getApiErrorMessage } from "@/lib/errors";
import { cn } from "@/lib/utils";
import { useId, useState } from "react";

import { DEPENDABOT_INTAKE_SETUP_COPY } from "./dependabotIntakeSetupCopy";
import { FirstRunHeading } from "./onboarding/first-run/FirstRunShell";

const COPY = DEPENDABOT_INTAKE_SETUP_COPY.import;

/** The intake source clamps the page size to 50. One repository rarely has more packages than this. */
const PACKAGE_LIMIT = 50;

interface DependabotIntakeImportStepProps {
  organizationId: string;
  factoryId: string;
  intakeId: string;
  onBusyChange: (busy: boolean) => void;
  onDone: () => void;
}

/**
 * Second Dependabot setup step. Lists the packages with open alerts that
 * match the intake filters and imports the ones the user picks. Nothing is
 * imported until the user asks for it, so a stale repository does not
 * flood the Backlog.
 */
export function DependabotIntakeImportStep(props: DependabotIntakeImportStepProps) {
  const items = useSearchFactoryIntakeItems({
    organizationId: props.organizationId,
    factoryId: props.factoryId,
    intakeId: props.intakeId,
    query: "",
    limit: PACKAGE_LIMIT,
  });
  const importItem = useImportFactoryIntakeItem(props.organizationId, props.factoryId);
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [progress, setProgress] = useState<{ done: number; total: number }>();
  const [error, setError] = useState<string>();

  const packages = (items.data ?? []).filter((item): item is FactoriesFactoryIntakeItem & { id: string } =>
    Boolean(item.id),
  );
  const importing = progress !== undefined;
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

  const importSelected = async () => {
    const ids = packages.map((item) => item.id).filter((id) => selected.has(id));
    if (ids.length === 0) return;
    setError(undefined);
    props.onBusyChange(true);
    try {
      for (const [index, id] of ids.entries()) {
        setProgress({ done: index, total: ids.length });
        await importItem.mutateAsync({ intakeId: props.intakeId, itemId: id });
        setSelected((current) => {
          const next = new Set(current);
          next.delete(id);
          return next;
        });
      }
      props.onDone();
    } catch (cause) {
      setError(getApiErrorMessage(cause, COPY.importError));
    } finally {
      setProgress(undefined);
      props.onBusyChange(false);
    }
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
            packageCount={packages.length}
            allSelected={allSelected}
            disabled={importing}
            onToggleAll={toggleAll}
          />
          {packages.length > 0 ? (
            <ul className="mt-3 space-y-2" data-testid="dependabot-import-packages">
              {packages.map((item) => (
                <PackageRow
                  key={item.id}
                  item={item}
                  checked={selected.has(item.id)}
                  disabled={importing}
                  onToggle={() => toggle(item.id)}
                />
              ))}
            </ul>
          ) : null}
        </div>

        {error ? (
          <p className="workspace-body-text text-destructive" role="alert">
            {error}
          </p>
        ) : null}

        <Button
          type="button"
          className="w-full"
          disabled={importing || selected.size === 0}
          onClick={() => void importSelected()}
          data-testid="dependabot-import-selected"
        >
          {progress ? COPY.importing(progress.done + 1, progress.total) : COPY.importSelected(selected.size)}
        </Button>
        <Button
          type="button"
          variant="ghost"
          className="w-full"
          disabled={importing}
          onClick={props.onDone}
          data-testid="dependabot-import-skip"
        >
          {COPY.skip}
        </Button>
      </div>
    </>
  );
}

function PackageListHeader({
  items,
  packageCount,
  allSelected,
  disabled,
  onToggleAll,
}: {
  items: ReturnType<typeof useSearchFactoryIntakeItems>;
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
    return (
      <div className="flex items-center justify-between gap-3">
        <p className="text-[13px] text-destructive" role="alert">
          {COPY.loadError}
        </p>
        <Button type="button" variant="outline" size="sm" onClick={() => void items.refetch()}>
          {COPY.retry}
        </Button>
      </div>
    );
  }
  if (packageCount === 0) {
    return (
      <p className="text-[13px] text-muted-foreground" data-testid="dependabot-import-empty">
        {COPY.empty}
      </p>
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
    <li
      className={cn(
        "flex items-center gap-3 rounded-lg border px-3 py-2.5 transition-colors",
        checked ? "border-foreground/20 bg-accent/50" : "border-border bg-card hover:border-foreground/15",
      )}
    >
      <Checkbox id={id} checked={checked} disabled={disabled} onChange={onToggle} />
      <Label htmlFor={id} className="flex min-w-0 flex-1 cursor-pointer items-center justify-between gap-3">
        <span className="truncate text-[13px] font-medium tracking-[-0.01em] text-foreground">{item.title}</span>
        {item.key ? <span className="shrink-0 text-[12px] text-muted-foreground">{item.key}</span> : null}
      </Label>
    </li>
  );
}
