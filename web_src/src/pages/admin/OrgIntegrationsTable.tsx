import { Heading } from "@/components/Heading/heading";
import { Text } from "@/components/Text/text";
import { Timestamp } from "@/components/Timestamp";
import { useQuery } from "@tanstack/react-query";
import { Plug } from "lucide-react";

export type AdminIntegration = {
  id: string;
  app_name: string;
  installation_name: string;
  state: string;
  state_description: string;
  details: Record<string, string>;
  created_at: string;
  updated_at: string;
};

type AdminIntegrationsResponse = {
  items: AdminIntegration[];
};

const LOAD_ERROR = "Could not load connections.";

const DETAIL_LABELS: Record<string, string> = {
  external_organization: "External organization",
  installation_uuid: "Installation UUID",
  installation_id: "Installation ID",
  hosted_app: "Hosted app",
};

const STATE_CLASSES: Record<string, string> = {
  ready: "bg-green-50 text-green-700 dark:bg-green-900/30 dark:text-green-300",
  pending: "bg-amber-50 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300",
  error: "bg-red-50 text-red-700 dark:bg-red-900/30 dark:text-red-300",
};

async function fetchOrgIntegrations(orgId: string): Promise<AdminIntegrationsResponse> {
  const response = await fetch(`/admin/api/organizations/${orgId}/integrations`, { credentials: "include" });
  if (!response.ok) {
    throw new Error(LOAD_ERROR);
  }
  return (await response.json()) as AdminIntegrationsResponse;
}

export function OrgIntegrationsTable({ orgId }: { orgId: string }) {
  const { data, isLoading, isError } = useQuery({
    queryKey: ["admin", "organizations", orgId, "integrations"],
    queryFn: () => fetchOrgIntegrations(orgId),
  });
  const items = data?.items ?? [];

  return (
    <div className="mb-8">
      <div className="flex items-center gap-2 mb-3">
        <Plug size={16} className="text-gray-600 dark:text-gray-400" />
        <Heading level={2} className="text-gray-800 text-base dark:text-gray-100">
          Connections ({items.length})
        </Heading>
      </div>
      <OrgIntegrationsBody isLoading={isLoading} isError={isError} items={items} />
    </div>
  );
}

function OrgIntegrationsBody({
  isLoading,
  isError,
  items,
}: {
  isLoading: boolean;
  isError: boolean;
  items: AdminIntegration[];
}) {
  if (isLoading) {
    return <Text className="text-gray-500 text-sm dark:text-gray-400">Loading...</Text>;
  }
  if (isError) {
    return <Text className="text-red-600 text-sm dark:text-red-400">{LOAD_ERROR}</Text>;
  }
  if (items.length === 0) {
    return <Text className="text-gray-500 text-sm dark:text-gray-400">This organization has no connections.</Text>;
  }

  return (
    <div className="bg-white rounded-md shadow-sm outline outline-slate-950/10 overflow-hidden dark:bg-gray-900 dark:outline-gray-700/70">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-slate-100 dark:border-gray-700/70">
            <th className="text-left px-4 py-2.5 text-gray-500 font-medium dark:text-gray-400">Connection</th>
            <th className="text-left px-4 py-2.5 text-gray-500 font-medium dark:text-gray-400">Status</th>
            <th className="text-left px-4 py-2.5 text-gray-500 font-medium dark:text-gray-400">Details</th>
            <th className="text-left px-4 py-2.5 text-gray-500 font-medium dark:text-gray-400">Updated</th>
          </tr>
        </thead>
        <tbody>
          {items.map((item) => (
            <tr key={item.id} className="border-b border-slate-50 last:border-0 align-top dark:border-gray-800/70">
              <td className="px-4 py-2.5">
                <div className="text-gray-800 font-medium dark:text-gray-100">{item.installation_name}</div>
                <div className="text-gray-500 text-xs dark:text-gray-400">{item.app_name}</div>
                <div className="font-mono text-xs text-gray-400 break-all dark:text-gray-500">{item.id}</div>
              </td>
              <td className="px-4 py-2.5">
                <span
                  className={`inline-block rounded px-2 py-0.5 text-xs font-medium ${
                    STATE_CLASSES[item.state] ?? "bg-slate-100 text-gray-700 dark:bg-gray-800 dark:text-gray-300"
                  }`}
                >
                  {item.state || "unknown"}
                </span>
                {item.state_description ? (
                  <div className="mt-1 max-w-sm text-xs text-gray-500 dark:text-gray-400">{item.state_description}</div>
                ) : null}
              </td>
              <td className="px-4 py-2.5">
                <OrgIntegrationDetails details={item.details} />
              </td>
              <td className="px-4 py-2.5">
                {item.updated_at ? (
                  <Timestamp
                    date={item.updated_at}
                    display="relative"
                    className="whitespace-nowrap text-gray-600 dark:text-gray-400"
                  />
                ) : (
                  "—"
                )}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function OrgIntegrationDetails({ details }: { details: Record<string, string> }) {
  const entries = Object.entries(details ?? {});
  if (entries.length === 0) {
    return <span className="text-gray-400 dark:text-gray-500">—</span>;
  }
  return (
    <dl className="space-y-1 text-xs">
      {entries.map(([key, value]) => (
        <div key={key}>
          <dt className="inline text-gray-500 dark:text-gray-400">{DETAIL_LABELS[key] ?? key}: </dt>
          <dd className="inline font-mono break-all text-gray-800 dark:text-gray-100">{value}</dd>
        </div>
      ))}
    </dl>
  );
}
