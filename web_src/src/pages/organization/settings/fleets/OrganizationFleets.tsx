import { useEffect, useRef, useState } from "react";
import { Link, useParams } from "react-router";
import {
  organizationsTerminateFleetRunner,
  type SuperplaneOrganizationsRunner,
  type SuperplaneOrganizationsFleet,
} from "@/api-client";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Label } from "@/components/ui/label";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import {
  useOrganizationFleetAccess,
  useOrganizationFleet,
  useOrganizationFleetCapacity,
  useOrganizationFleetRunners,
  useOrganizationFleetRunner,
  useRefreshOrganizationFleets,
} from "@/hooks/useOrganizationFleetManagement";
import {
  PageControls,
  RelativeTimestamp,
  StateBadge,
  headerCellClass,
  rowClass,
  tableClass,
} from "@/pages/admin/fleetTable";
import { FleetForm } from "./FleetForm";
import { ConnectRunner } from "./ConnectRunner";

function LoadError({ retry, subject }: { retry: () => void; subject: string }) {
  return (
    <div role="alert" className="space-y-2">
      <p>Could not load {subject}. Check access and retry.</p>
      <Button variant="outline" onClick={retry}>
        Retry
      </Button>
    </div>
  );
}

function Capacity({ organizationId, fleetId }: { organizationId: string; fleetId: string }) {
  const capacity = useOrganizationFleetCapacity(organizationId, fleetId);
  if (capacity.isError) return <LoadError subject="capacity" retry={() => void capacity.refetch()} />;
  if (!capacity.data) return <p>Loading capacity…</p>;
  return (
    <dl className="grid grid-cols-2 gap-3 text-sm">
      {(
        [
          ["Runnable tasks", capacity.data.runnableTasks],
          ["Pending runners", capacity.data.pendingRunners],
          ["Idle runners", capacity.data.idleRunners],
          ["Busy runners", capacity.data.busyRunners],
          ["Terminated runners", capacity.data.terminatedRunners],
        ] as const
      ).map(([label, count]) => (
        <div key={label}>
          <dt className="text-muted-foreground">{label}</dt>
          <dd>{Number(count ?? 0)}</dd>
        </div>
      ))}
    </dl>
  );
}

export function OrganizationFleets({ organizationId, basePath }: { organizationId: string; basePath: string }) {
  const access = useOrganizationFleetAccess(organizationId);
  const refresh = useRefreshOrganizationFleets(organizationId);
  const [creating, setCreating] = useState(false);
  const catalog = access.catalog;
  return (
    <div className="space-y-5">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-medium">Organization fleets</h2>
        {access.canConfigure && <Button onClick={() => setCreating(true)}>Create fleet</Button>}
      </div>
      <AccessNotice organizationId={organizationId} />
      {creating && access.canConfigure && (
        <FleetForm
          organizationId={organizationId}
          onCancel={() => setCreating(false)}
          onSaved={() => setCreating(false)}
          refresh={refresh}
        />
      )}
      {catalog.isError ? (
        <LoadError subject="fleets" retry={() => void catalog.refetch()} />
      ) : !catalog.data ? (
        <p>Loading fleets…</p>
      ) : catalog.data.length === 0 ? (
        <p>No organization fleets. An administrator can create a fleet when organization runners are enabled.</p>
      ) : (
        <div className={tableClass}>
          <table className="w-full text-sm">
            <thead>
              <tr>
                {["Fleet ID", "Enabled", "Operating system", "Architecture", "Runner version", "Capacity"].map(
                  (label) => (
                    <th key={label} className={headerCellClass}>
                      {label}
                    </th>
                  ),
                )}
              </tr>
            </thead>
            <tbody>
              {catalog.data.map((fleet) => (
                <tr key={fleet.id} className={rowClass}>
                  <td className="p-4">
                    <Link className="text-blue-600 underline" to={`${basePath}/${encodeURIComponent(fleet.id ?? "")}`}>
                      {fleet.id}
                    </Link>
                  </td>
                  <td className="p-4">{fleet.enabled ? "Enabled" : "Disabled"}</td>
                  <td className="p-4">{fleet.spec?.operatingSystem || "Unavailable"}</td>
                  <td className="p-4">{fleet.spec?.architecture || "Unavailable"}</td>
                  <td className="p-4">{fleet.runnerVersion || "Unavailable"}</td>
                  <td className="p-4">
                    <Capacity organizationId={organizationId} fleetId={fleet.id!} />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}

function AccessNotice({ organizationId }: { organizationId: string }) {
  const access = useOrganizationFleetAccess(organizationId);
  const refresh = useRefreshOrganizationFleets(organizationId);
  if (access.accessError) return <LoadError subject="fleet configuration access" retry={refresh} />;
  if (!access.resolved) return <p>Checking fleet configuration access…</p>;
  if (!access.enabled)
    return (
      <p role="status">
        Organization runners are disabled. Existing fleets remain available for inspection and runner termination.
      </p>
    );
  return null;
}

export function OrganizationFleetDetails({ organizationId, basePath }: { organizationId: string; basePath: string }) {
  const { fleetId = "" } = useParams();
  return (
    <FleetDetails
      key={`${organizationId}/${fleetId}`}
      organizationId={organizationId}
      fleetId={fleetId}
      basePath={basePath}
    />
  );
}

function FleetDetails({
  organizationId,
  fleetId,
  basePath,
}: {
  organizationId: string;
  fleetId: string;
  basePath: string;
}) {
  const fleet = useOrganizationFleet(organizationId, fleetId);
  const access = useOrganizationFleetAccess(organizationId);
  const refresh = useRefreshOrganizationFleets(organizationId);
  const [editing, setEditing] = useState(false);
  const [connecting, setConnecting] = useState(false);
  useEffect(() => {
    if (!access.canConfigure) {
      setEditing(false);
      setConnecting(false);
    }
  }, [access.canConfigure]);
  if (fleet.isError) return <LoadError subject="fleet" retry={() => void fleet.refetch()} />;
  if (!fleet.data) return <p>Loading fleet…</p>;
  const record = fleet.data;
  return (
    <div className="space-y-5">
      <Link to={basePath} className="text-blue-600 underline">
        Back to Fleets
      </Link>
      <h2 className="text-lg font-medium">{fleetId}</h2>
      <AccessNotice organizationId={organizationId} />
      <div className="flex gap-2">
        {access.canConfigure && (
          <Button variant="outline" onClick={() => setEditing(true)}>
            Edit fleet
          </Button>
        )}
        {access.canConfigure && record.enabled && <Button onClick={() => setConnecting(true)}>Connect runner</Button>}
      </div>
      {!record.enabled && (
        <p>This fleet is disabled. New work and runner registration stop. Active work and cleanup remain available.</p>
      )}
      {editing && access.canConfigure ? (
        <FleetForm
          organizationId={organizationId}
          fleet={record}
          onCancel={() => setEditing(false)}
          onSaved={() => setEditing(false)}
          refresh={refresh}
        />
      ) : (
        <FleetFields fleet={record} />
      )}
      <h3 className="font-medium">Capacity</h3>
      <Capacity organizationId={organizationId} fleetId={fleetId} />
      <FleetRunners organizationId={organizationId} fleetId={fleetId} basePath={basePath} />
      {connecting && access.canConfigure && record.enabled && (
        <ConnectRunner
          organizationId={organizationId}
          fleetId={fleetId}
          onClose={() => setConnecting(false)}
          refresh={refresh}
        />
      )}
    </div>
  );
}

function FleetFields({ fleet }: { fleet: SuperplaneOrganizationsFleet }) {
  const spec = fleet.spec ?? {};
  const fields = {
    "Fleet ID": fleet.id,
    Enabled: fleet.enabled ? "Yes" : "No",
    "Runner version": fleet.runnerVersion,
    "Operating system": spec.operatingSystem,
    Architecture: spec.architecture,
    "CPU millicores": spec.cpuMillicores,
    "Memory MB": spec.memoryMb,
    "Disk GB": spec.diskGb,
    Capabilities: spec.capabilities?.join(", "),
  };
  return (
    <dl className="grid grid-cols-2 gap-3 text-sm">
      {Object.entries(fields).map(([label, value]) => (
        <div key={label}>
          <dt className="text-muted-foreground">{label}</dt>
          <dd>{value ?? "Unavailable"}</dd>
        </div>
      ))}
    </dl>
  );
}

function FleetRunners({
  organizationId,
  fleetId,
  basePath,
}: {
  organizationId: string;
  fleetId: string;
  basePath: string;
}) {
  const [state, setState] = useState("all");
  const [cursors, setCursors] = useState<string[]>([]);
  const runners = useOrganizationFleetRunners(organizationId, fleetId, state, cursors.at(-1));
  const rows = runners.data?.runners ?? [];
  return (
    <section className="space-y-3">
      <h3 className="font-medium">Runners</h3>
      <Label htmlFor="runner-state">State</Label>
      <Select
        value={state}
        onValueChange={(value) => {
          setState(value);
          setCursors([]);
        }}
      >
        <SelectTrigger id="runner-state">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {["all", "pending", "idle", "busy", "terminated"].map((value) => (
            <SelectItem key={value} value={value}>
              {value === "all" ? "All states" : value}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      {runners.isError ? (
        <LoadError subject="runners" retry={() => void runners.refetch()} />
      ) : !runners.data ? (
        <p>Loading runners…</p>
      ) : (
        <>
          {rows.length === 0 ? (
            <p>No runners match this filter. Choose another state or connect a runner.</p>
          ) : (
            <div className={tableClass}>
              <table className="w-full text-sm">
                <thead>
                  <tr>
                    {["Runner ID", "State", "Version", "Host", "Last seen"].map((label) => (
                      <th key={label} className={headerCellClass}>
                        {label}
                      </th>
                    ))}
                  </tr>
                </thead>
                <tbody>
                  {rows.map((runner) => (
                    <tr key={runner.id} className={rowClass}>
                      <td className="p-4">
                        <Link
                          className="text-blue-600 underline"
                          to={`${basePath}/${encodeURIComponent(fleetId)}/runners/${encodeURIComponent(runner.id!)}`}
                        >
                          {runner.id}
                        </Link>
                      </td>
                      <td className="p-4">
                        <StateBadge state={runner.state ?? "Unavailable"} />
                      </td>
                      <td className="p-4">{runner.runnerVersion || "Unavailable"}</td>
                      <td className="p-4">
                        {runner.hostname || "Unavailable"}
                        <br />
                        {runner.ip || "Unavailable"}
                        <br />
                        {runner.os || "Unavailable"} / {runner.arch || "Unavailable"}
                      </td>
                      <td className="p-4">
                        <RelativeTimestamp value={runner.lastSeenAt} />
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
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
    </section>
  );
}

export function OrganizationRunnerDetails({ organizationId, basePath }: { organizationId: string; basePath: string }) {
  const { fleetId = "", runnerId = "" } = useParams();
  return (
    <RunnerDetails
      key={`${organizationId}/${fleetId}/${runnerId}`}
      organizationId={organizationId}
      fleetId={fleetId}
      runnerId={runnerId}
      basePath={basePath}
    />
  );
}

function RunnerDetails({
  organizationId,
  fleetId,
  runnerId,
  basePath,
}: {
  organizationId: string;
  fleetId: string;
  runnerId: string;
  basePath: string;
}) {
  const runner = useOrganizationFleetRunner(organizationId, fleetId, runnerId);
  const access = useOrganizationFleetAccess(organizationId);
  const refresh = useRefreshOrganizationFleets(organizationId);
  const [confirming, setConfirming] = useState(false);
  const [requested, setRequested] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState("");
  const abort = useRef<AbortController | null>(null);
  useEffect(() => () => abort.current?.abort(), []);
  const terminate = async () => {
    if (submitting || !access.canTerminate) return;
    const request = new AbortController();
    abort.current = request;
    setSubmitting(true);
    setError("");
    try {
      const response = await organizationsTerminateFleetRunner(
        withOrganizationHeader({
          organizationId,
          path: { id: organizationId, runnerId },
          query: { fleetId, states: runner.data?.state ? [runner.data.state] : [] },
          signal: request.signal,
        }),
      );
      if (request.signal.aborted) return;
      setRequested(response.data.runner?.state !== "terminated");
      setConfirming(false);
    } catch {
      if (request.signal.aborted) return;
      setError("Could not terminate the runner. If it is busy and reusable, wait for the task to finish and retry.");
      setConfirming(false);
    } finally {
      if (!request.signal.aborted) {
        setSubmitting(false);
        refresh();
      }
    }
  };
  if (runner.isError) return <LoadError subject="runner" retry={() => void runner.refetch()} />;
  if (!runner.data) return <p>Loading runner…</p>;
  const record = runner.data;
  return (
    <div className="space-y-5">
      <Link className="text-blue-600 underline" to={`${basePath}/${encodeURIComponent(fleetId)}`}>
        Back to fleet
      </Link>
      <h2 className="text-lg font-medium">Runner {runnerId}</h2>
      <RunnerFields runner={record} />
      {record.state === "terminated" ? (
        <p role="status">Runner terminated. The host machine remains.</p>
      ) : requested ? (
        <p role="status">Termination requested. Waiting for the server to confirm termination.</p>
      ) : null}
      {error && <p role="alert">{error}</p>}
      {access.canTerminate && record.state !== "terminated" && (
        <Button variant="destructive" disabled={requested || submitting} onClick={() => setConfirming(true)}>
          Terminate runner
        </Button>
      )}
      {confirming && access.canTerminate && (
        <Dialog open onOpenChange={setConfirming}>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>Terminate runner?</DialogTitle>
              <DialogDescription>Termination does not delete the host machine.</DialogDescription>
            </DialogHeader>
            {record.state === "busy" && record.ephemeral && (
              <p>This request cancels active work. Termination completes asynchronously.</p>
            )}
            {record.state === "busy" && !record.ephemeral && (
              <p>A busy reusable runner cannot be terminated. Wait for its task to finish and retry.</p>
            )}
            <Button variant="destructive" disabled={submitting} onClick={() => void terminate()}>
              Confirm termination
            </Button>
            <Button variant="outline" onClick={() => setConfirming(false)}>
              Keep runner
            </Button>
          </DialogContent>
        </Dialog>
      )}
    </div>
  );
}

function RunnerFields({ runner }: { runner: SuperplaneOrganizationsRunner }) {
  const fields = {
    "Runner ID": runner.id,
    State: runner.state,
    "Runner version": runner.runnerVersion,
    "Lifecycle mode": runner.ephemeral ? "Ephemeral" : "Reusable",
    "Operating system": runner.os,
    Architecture: runner.arch,
    Hostname: runner.hostname,
    "IP address": runner.ip,
    Tags: Object.entries(runner.tags ?? {})
      .map(([key, value]) => `${key}: ${value}`)
      .join(", "),
    Created: runner.createdAt,
    "Registration time": runner.registeredAt,
    "Last seen": runner.lastSeenAt,
    "Terminated at": runner.terminatedAt,
    "Termination reason": runner.terminationReason,
    "Provider deleted at": runner.providerDeletedAt,
  };
  return (
    <dl className="grid grid-cols-2 gap-3 text-sm">
      {Object.entries(fields).map(([label, value]) => (
        <div key={label}>
          <dt className="text-muted-foreground">{label}</dt>
          <dd className="break-all">{value || "Unavailable"}</dd>
        </div>
      ))}
    </dl>
  );
}
