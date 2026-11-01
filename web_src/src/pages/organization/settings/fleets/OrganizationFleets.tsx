import { useEffect, useState, type ReactNode } from "react";
import { Link, useParams } from "react-router";
import { Button } from "@/components/ui/button";
import { FleetCapacity, FleetStatus } from "./FleetIndicators";
import {
  fleetTableClass as tableClass,
  fleetHeaderCellClass as headerCellClass,
  fleetRowClass as rowClass,
  fleetLinkClass,
} from "./fleetStyles";
import { ChevronLeft } from "lucide-react";
import { FleetEnabledButton } from "./FleetEnabledButton";
import { FleetRunners } from "./FleetRunners";
import { LoadError } from "./LoadError";
import {
  useOrganizationFleetAccess,
  useOrganizationFleet,
  useOrganizationFleetCapacity,
  useRefreshOrganizationFleets,
} from "@/hooks/useOrganizationFleetManagement";
import { FleetForm } from "./FleetForm";
import { FleetSettings } from "./FleetSettings";
import { ConnectRunner } from "./ConnectRunner";

function Capacity({ organizationId, fleetId }: { organizationId: string; fleetId: string }) {
  const capacity = useOrganizationFleetCapacity(organizationId, fleetId);
  if (capacity.isError) return <LoadError subject="capacity" retry={() => void capacity.refetch()} />;
  if (!capacity.data) return <p>Loading capacity…</p>;
  return <FleetCapacity capacity={capacity.data} />;
}

export function OrganizationFleets({
  organizationId,
  basePath,
  renderPage,
}: {
  organizationId: string;
  basePath: string;
  renderPage?: (content: ReactNode, actions: ReactNode) => ReactNode;
}) {
  const access = useOrganizationFleetAccess(organizationId);
  const refresh = useRefreshOrganizationFleets(organizationId);
  const [creating, setCreating] = useState(false);
  useEffect(() => {
    if (!access.canConfigure) setCreating(false);
  }, [access.canConfigure]);
  const catalog = access.catalog;
  const actions =
    access.canConfigure && !creating ? (
      <Button size="sm" onClick={() => setCreating(true)}>
        Create fleet
      </Button>
    ) : null;
  const content = (
    <div className="space-y-5 text-[13px]">
      <AccessNotice organizationId={organizationId} />
      {creating && access.canConfigure ? (
        <FleetForm
          organizationId={organizationId}
          onCancel={() => setCreating(false)}
          onSaved={() => setCreating(false)}
          refresh={refresh}
        />
      ) : catalog.isError ? (
        <LoadError subject="fleets" retry={() => void catalog.refetch()} />
      ) : !catalog.data ? (
        <p>Loading fleets…</p>
      ) : catalog.data.length === 0 ? (
        <p>No organization fleets. An administrator can create a fleet when organization runners are enabled.</p>
      ) : (
        <div className={tableClass}>
          <table className="w-full text-[13px]">
            <thead>
              <tr>
                {["Fleet ID", "OS", "Runner version", "Capacity"].map((label) => (
                  <th key={label} className={headerCellClass}>
                    {label}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {catalog.data.map((fleet) => (
                <tr key={fleet.id} className={rowClass}>
                  <td className="p-4">
                    <div className="flex items-center gap-2">
                      <FleetStatus enabled={!!fleet.enabled} />
                      <Link className={fleetLinkClass} to={`${basePath}/${encodeURIComponent(fleet.id ?? "")}`}>
                        {fleet.id}
                      </Link>
                    </div>
                  </td>
                  <td className="p-4 text-muted-foreground">
                    {fleet.spec?.operatingSystem || "—"} / {fleet.spec?.architecture || "—"}
                  </td>
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
  return renderPage ? (
    renderPage(content, actions)
  ) : (
    <>
      <div className="mb-4 flex justify-end">{actions}</div>
      {content}
    </>
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
  const [connecting, setConnecting] = useState(false);
  useEffect(() => {
    if (!access.canConfigure) {
      setConnecting(false);
    }
  }, [access.canConfigure]);
  if (fleet.isError) return <LoadError subject="fleet" retry={() => void fleet.refetch()} />;
  if (!fleet.data) return <p>Loading fleet…</p>;
  const record = fleet.data;
  if (connecting && access.canConfigure && record.enabled)
    return (
      <ConnectRunner
        key={record.runnerVersion}
        organizationId={organizationId}
        fleet={record}
        onClose={() => setConnecting(false)}
        refresh={refresh}
      />
    );
  return (
    <div className="space-y-5 text-[13px]">
      <Link
        to={basePath}
        className="inline-flex items-center gap-1.5 text-[13px] text-muted-foreground hover:text-foreground"
      >
        <ChevronLeft className="size-3.5" aria-hidden="true" />
        Back to Fleets
      </Link>
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex items-center gap-2">
          <FleetStatus enabled={!!record.enabled} />
          <h2 className="text-base font-medium">{fleetId}</h2>
        </div>
        {access.canConfigure && (
          <div className="flex items-center gap-2">
            <FleetEnabledButton organizationId={organizationId} fleet={record} refresh={refresh} />
            {record.enabled && (
              <Button size="sm" onClick={() => setConnecting(true)}>
                Connect runner
              </Button>
            )}
          </div>
        )}
      </div>
      {!record.enabled && (
        <p>This fleet is disabled. New work and runner registration stop. Active work and cleanup remain available.</p>
      )}
      <FleetSettings
        organizationId={organizationId}
        fleet={record}
        canConfigure={access.canConfigure}
        refresh={refresh}
      />
      <FleetRunners organizationId={organizationId} fleetId={fleetId} />
    </div>
  );
}
