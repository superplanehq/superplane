import { Button } from "@/components/ui/button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { IntegrationIcon } from "@/ui/componentSidebar/integrationIcons";
import { getIntegrationTypeDisplayName } from "@/lib/integrationDisplayName";

import { HomeIntegrationConnectRow, StatusDot } from "./HomeIntegrationConnectRow";
import {
  resolveHomeIntegrationStatus,
  selectionFromInstance,
  type IntegrationInstanceSummary,
  type IntegrationSelections,
} from "./homeIntegrationStatus";
import { useIntegrationConnectDialog } from "./useIntegrationConnectDialog";

export type { IntegrationSelections };

export function IntegrationsSection({
  integrations,
  organizationId,
  selections,
  onSelectionsChange,
  variant = "status",
}: {
  integrations: string[];
  organizationId: string;
  selections: IntegrationSelections;
  onSelectionsChange: (selections: IntegrationSelections) => void;
  /** `status` shows Connected / Pending rows with switcher; `select` keeps the legacy instance picker. */
  variant?: "select" | "status";
}) {
  const connect = useIntegrationConnectDialog({
    organizationId,
    integrationNames: integrations,
    selections,
    onSelectionsChange,
  });

  return (
    <>
      <IntegrationList
        integrationData={connect.integrationData}
        variant={variant}
        selections={selections}
        onSelectionsChange={onSelectionsChange}
        onConnect={connect.requestConnect}
        onConfigure={connect.configure}
        onCreateNew={connect.createNew}
      />
      {connect.dialogs}
    </>
  );
}

function IntegrationList({
  integrationData,
  variant,
  selections,
  onSelectionsChange,
  onConnect,
  onConfigure,
  onCreateNew,
}: {
  integrationData: IntegrationInstanceSummary[];
  variant: "select" | "status";
  selections: IntegrationSelections;
  onSelectionsChange: (selections: IntegrationSelections) => void;
  onConnect: (name: string) => void;
  onConfigure: (id: string) => void;
  onCreateNew: (name: string) => void;
}) {
  return (
    <div className="divide-y divide-slate-200 rounded-md border border-slate-200 dark:divide-gray-700/70 dark:border-gray-700/70">
      {integrationData.map((data) =>
        variant === "status" ? (
          <HomeIntegrationConnectRow
            key={data.name}
            name={data.name}
            status={resolveHomeIntegrationStatus(data, selections[data.name]?.id)}
            instances={data.allInstances}
            selectedId={selections[data.name]?.id}
            selectedName={selections[data.name]?.name}
            onConnect={() => onConnect(data.name)}
            onConfigure={onConfigure}
            onSelect={(id, instanceName) => {
              const instance = data.allInstances.find((item) => item.metadata?.id === id);
              const selection = instance ? selectionFromInstance(instance) : null;
              onSelectionsChange({
                ...selections,
                [data.name]: selection ?? { id, name: instanceName, ready: false },
              });
            }}
            onCreateNew={() => onCreateNew(data.name)}
          />
        ) : (
          <IntegrationRow
            key={data.name}
            data={data}
            selectedId={selections[data.name]?.id}
            onSelect={(id, name) => {
              const instance = data.allInstances.find((item) => item.metadata?.id === id);
              const selection = instance ? selectionFromInstance(instance) : null;
              onSelectionsChange({
                ...selections,
                [data.name]: selection ?? { id, name, ready: false },
              });
            }}
            onConfigure={onConfigure}
            onCreateNew={() => onCreateNew(data.name)}
          />
        ),
      )}
    </div>
  );
}

function IntegrationRow({
  data,
  selectedId,
  onSelect,
  onConfigure,
  onCreateNew,
}: {
  data: IntegrationInstanceSummary;
  selectedId?: string;
  onSelect: (id: string, name: string) => void;
  onConfigure: (id: string) => void;
  onCreateNew: () => void;
}) {
  const displayName =
    getIntegrationTypeDisplayName(undefined, data.name) || data.name.charAt(0).toUpperCase() + data.name.slice(1);

  const handleInstanceSelect = (instanceId: string) => {
    const instance = data.allInstances.find((i) => i.metadata?.id === instanceId);
    if (!instance?.metadata?.id) return;
    if (instance.status?.state !== "ready") {
      onConfigure(instance.metadata.id);
      return;
    }
    if (instance.metadata.name) {
      onSelect(instance.metadata.id, instance.metadata.name);
    }
  };

  return (
    <div className="flex min-h-7 items-center gap-2 px-3 py-2.5">
      <IntegrationIcon integrationName={data.name} className="h-4 w-4 shrink-0" size={16} />
      <span className="shrink-0 truncate text-sm font-medium text-slate-900 dark:text-gray-100">{displayName}</span>
      {data.allInstances.length > 0 ? (
        <>
          <Select value={selectedId || ""} onValueChange={handleInstanceSelect}>
            <SelectTrigger className="h-7 min-w-0 flex-1 text-xs">
              <SelectValue placeholder={`Select ${displayName}`} />
            </SelectTrigger>
            <SelectContent>
              {data.allInstances.map((instance) => (
                <SelectItem key={instance.metadata?.id} value={instance.metadata?.id ?? ""}>
                  <span className="flex items-center gap-1.5">
                    <span>{instance.metadata?.name || instance.metadata?.id}</span>
                    <StatusDot state={instance.status?.state} />
                  </span>
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <Button
            type="button"
            variant="link"
            size="xs"
            onClick={onCreateNew}
            className="h-auto shrink-0 p-0 text-xs font-normal"
          >
            or create new
          </Button>
        </>
      ) : (
        <>
          <span className="min-w-0 flex-1 text-xs font-medium text-gray-400 dark:text-gray-500">Not connected</span>
          <Button type="button" variant="outline" size="xs" className="shrink-0" onClick={onCreateNew}>
            Connect
          </Button>
        </>
      )}
    </div>
  );
}
