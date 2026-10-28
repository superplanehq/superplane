import { useQuery } from "@tanstack/react-query";
import { useState } from "react";

import { meListModelStatus, type MeModelStatusCell } from "@/api-client";
import { Button } from "@/components/ui/button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { usePageTitle } from "@/hooks/usePageTitle";
import {
  MODEL_STATUS_DEFAULT_WINDOW,
  MODEL_STATUS_PROVIDERS,
  MODEL_STATUS_WINDOWS,
  type ModelStatusBand,
  type ModelStatusMatrixCell,
  type ModelStatusWindow,
  isModelStatusWindow,
  modelStatusCallCountLabel,
  modelStatusMatrixRows,
  modelStatusProviderLabel,
  modelStatusWindowLabel,
} from "@/lib/modelStatus";
import { cn } from "@/lib/utils";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";

import { FactorySettingsCard, FactorySettingsPageFrame } from "./FactorySettingsCard";
import { useFactorySettingsLayout } from "./factorySettingsLayoutContext";
import { ORGANIZATION_MODEL_STATUS_COPY as COPY } from "./organizationModelStatusCopy";

const BAND_CLASSNAME: Record<ModelStatusBand, string> = {
  available: "bg-green-100 text-green-900 dark:bg-green-950/60 dark:text-green-200",
  degraded: "bg-amber-100 text-amber-950 dark:bg-amber-950/60 dark:text-amber-200",
  unavailable: "bg-red-100 text-red-900 dark:bg-red-950/60 dark:text-red-200",
  "no-data": "bg-muted text-muted-foreground",
};

const BAND_LABEL: Record<ModelStatusBand, string> = {
  available: COPY.available,
  degraded: COPY.degraded,
  unavailable: COPY.unavailable,
  "no-data": COPY.noData,
};

export function FactorySettingsOrganizationModelStatusPage() {
  const { organizationId, factory } = useFactorySettingsLayout();
  const [window, setWindow] = useState<ModelStatusWindow>(MODEL_STATUS_DEFAULT_WINDOW);
  const query = useQuery({
    queryKey: ["me", "model-status", window],
    queryFn: async () => {
      const response = await meListModelStatus(
        withOrganizationHeader({
          organizationId,
          query: { window },
        }),
      );
      if (!response.data) {
        throw new Error(COPY.loadError);
      }
      return response.data;
    },
    enabled: Boolean(organizationId),
    refetchInterval: 30_000,
  });

  usePageTitle([COPY.pageTitle, "Settings", factory.name ?? "Workspace"]);

  return (
    <FactorySettingsPageFrame
      title={COPY.pageTitle}
      subtitle={COPY.pageSubtitle}
      wide
      actions={
        <label className="flex items-center gap-2 text-sm">
          <span className="text-muted-foreground">{COPY.windowLabel}</span>
          <Select
            value={window}
            onValueChange={(value) => {
              if (isModelStatusWindow(value)) {
                setWindow(value);
              }
            }}
          >
            <SelectTrigger aria-label={COPY.windowLabel} className="w-[9.5rem]" data-testid="model-status-window">
              <SelectValue>{modelStatusWindowLabel(window)}</SelectValue>
            </SelectTrigger>
            <SelectContent>
              {MODEL_STATUS_WINDOWS.map((option) => (
                <SelectItem key={option} value={option}>
                  {modelStatusWindowLabel(option)}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </label>
      }
    >
      <FactorySettingsCard description={COPY.legend} data-testid="model-status-page">
        {query.isPending ? <p role="status">{COPY.loading}</p> : null}
        {query.isError ? (
          <div className="flex flex-col items-start gap-3">
            <p>{COPY.loadError}</p>
            <Button variant="outline" onClick={() => void query.refetch()}>
              {COPY.tryAgain}
            </Button>
          </div>
        ) : null}
        {query.data ? <ModelStatusMatrix cells={query.data.cells ?? []} /> : null}
      </FactorySettingsCard>
    </FactorySettingsPageFrame>
  );
}

export function ModelStatusMatrix({ cells }: { cells: MeModelStatusCell[] }) {
  const rows = modelStatusMatrixRows(cells);
  if (rows.length === 0) {
    return <p>{COPY.empty}</p>;
  }

  return (
    <div className="overflow-x-auto">
      <table className="w-full min-w-[36rem] border-collapse text-left text-sm" data-testid="model-status-matrix">
        <caption className="sr-only">{COPY.pageTitle}</caption>
        <thead>
          <tr className="border-b border-border">
            <th scope="col" className="px-3 py-2 font-medium">
              {COPY.modelColumn}
            </th>
            {MODEL_STATUS_PROVIDERS.map((provider) => (
              <th key={provider} scope="col" className="px-3 py-2 font-medium">
                {modelStatusProviderLabel(provider)}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {rows.map((row) => (
            <tr
              key={row.model}
              className="border-b border-border last:border-0"
              data-testid={`model-status-row-${row.model}`}
            >
              <th scope="row" className="px-3 py-2 font-medium">
                {row.model}
              </th>
              {MODEL_STATUS_PROVIDERS.map((provider) => (
                <td key={provider} className="px-2 py-2">
                  <ModelStatusCellView cell={row.cells[provider]} />
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function ModelStatusCellView({ cell }: { cell: ModelStatusMatrixCell }) {
  const label =
    cell.band === "no-data"
      ? `${modelStatusProviderLabel(cell.provider)} ${cell.model}: ${COPY.noData}`
      : `${modelStatusProviderLabel(cell.provider)} ${cell.model}: ${cell.score}, ${modelStatusCallCountLabel(cell.callCount)}, ${BAND_LABEL[cell.band]}`;

  return (
    <div
      className={cn("rounded-md px-3 py-2", BAND_CLASSNAME[cell.band])}
      data-testid={`model-status-cell-${cell.provider}-${cell.model}`}
      aria-label={label}
    >
      {cell.band === "no-data" ? (
        <p>{COPY.noData}</p>
      ) : (
        <>
          <p className="font-medium">{cell.score}</p>
          <p className="text-xs">{modelStatusCallCountLabel(cell.callCount)}</p>
        </>
      )}
    </div>
  );
}
