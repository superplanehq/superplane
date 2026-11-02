import type { SuperplaneOrganizationsRunner } from "@/api-client";
import { useState } from "react";
import { CopyButton } from "@/ui/CopyButton";
import { TerminateRunnerButton } from "./TerminateRunnerButton";
import { Tabs, TabsContent } from "@/components/ui/tabs";
import {
  useOrganizationFleetCapacity,
  useOrganizationFleetRunners,
  useOrganizationFleetAccess,
} from "@/hooks/useOrganizationFleetManagement";
import { PageControls, RelativeTimestamp } from "@/pages/admin/fleetTable";
import { FleetRunnerTabs } from "./FleetIndicators";
import { LoadError } from "./LoadError";
import {
  fleetTableClass as tableClass,
  fleetHeaderCellClass as headerCellClass,
  fleetRowClass as rowClass,
} from "./fleetStyles";

const emptyMessages: Record<string, string> = {
  pending: "No runners are waiting to register.",
  idle: "No idle runners are available.",
  busy: "No runners are executing work.",
  terminated: "No runners have terminated.",
};

export function FleetRunners({ organizationId, fleetId }: { organizationId: string; fleetId: string }) {
  const access = useOrganizationFleetAccess(organizationId);
  const [state, setState] = useState("busy");
  const [cursors, setCursors] = useState<string[]>([]);
  const runners = useOrganizationFleetRunners(organizationId, fleetId, state, cursors.at(-1));
  const capacity = useOrganizationFleetCapacity(organizationId, fleetId);
  const rows = runners.data?.runners ?? [];
  return (
    <section className="space-y-3">
      <h3 className="font-medium">Runners</h3>
      <Tabs
        value={state}
        onValueChange={(value) => {
          setState(value);
          setCursors([]);
        }}
      >
        <FleetRunnerTabs capacity={capacity.data} />
        <TabsContent value={state} className="pt-3">
          {capacity.isError && <LoadError subject="runner counts" retry={() => void capacity.refetch()} />}
          {runners.isError ? (
            <LoadError subject="runners" retry={() => void runners.refetch()} />
          ) : !runners.data ? (
            <p>Loading runners…</p>
          ) : (
            <>
              {rows.length === 0 ? (
                <p className="py-6 text-[13px] text-muted-foreground">{emptyMessages[state]}</p>
              ) : (
                <RunnerTable
                  runners={rows}
                  organizationId={organizationId}
                  fleetId={fleetId}
                  canTerminate={access.canTerminate}
                />
              )}
              <PageControls
                pageIndex={cursors.length}
                rowCount={rows.length}
                totalCount={Number(runners.data.totalCount ?? 0)}
                hasNextPage={!!runners.data.hasNextPage}
                onPrevious={() => setCursors(cursors.slice(0, -1))}
                onNext={() => {
                  const lastId = rows.at(-1)?.id;
                  if (lastId && runners.data?.hasNextPage) setCursors([...cursors, lastId]);
                }}
                previousTestId="organization-runners-previous"
                nextTestId="organization-runners-next"
              />
            </>
          )}
        </TabsContent>
      </Tabs>
    </section>
  );
}

function RunnerTable({
  organizationId,
  canTerminate,
  runners,
  fleetId,
}: {
  organizationId: string;
  canTerminate: boolean;
  runners: SuperplaneOrganizationsRunner[];
  fleetId: string;
}) {
  return (
    <div className={tableClass}>
      <table className="w-full text-[13px]">
        <thead>
          <tr>
            {["Runner ID", "Version", "Host", "Last seen"].map((label) => (
              <th key={label} className={headerCellClass}>
                {label}
              </th>
            ))}
            {canTerminate && (
              <th className={headerCellClass}>
                <span className="sr-only">Actions</span>
              </th>
            )}
          </tr>
        </thead>
        <tbody>
          {runners.map((runner) => (
            <tr key={runner.id} className={rowClass}>
              <td className="px-4 py-3">
                <div className="flex items-center gap-1">
                  <code className="font-mono text-[11px] tracking-tight text-muted-foreground">{runner.id}</code>
                  {runner.id && <CopyButton text={runner.id} ariaLabel="Copy runner ID" />}
                </div>
              </td>
              <td className="px-4 py-3">{runner.runnerVersion || "Unavailable"}</td>
              <td className="px-4 py-3">
                {runner.hostname || "Unavailable"}
                <br />
                {runner.ip || "Unavailable"}
                <br />
                {runner.os || "Unavailable"} / {runner.arch || "Unavailable"}
              </td>
              <td className="px-4 py-3">
                <RelativeTimestamp value={runner.lastSeenAt} />
              </td>
              {canTerminate && (
                <td className="px-4 py-3 text-right">
                  <TerminateRunnerButton organizationId={organizationId} fleetId={fleetId} runner={runner} />
                </td>
              )}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
