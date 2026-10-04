import { Heading } from "@/components/Heading/heading";
import { Text } from "@/components/Text/text";
import { Timestamp } from "@/components/Timestamp";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Plug, Search } from "lucide-react";
import { useCallback, useEffect, useRef, useState } from "react";
import AdminPagination from "./AdminPagination";

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
  total: number;
};

const PAGE_SIZE = 50;
const LOAD_ERROR = "Could not load connections.";

const DETAIL_LABELS: Record<string, string> = {
  external_organization: "External organization",
  installation_uuid: "Installation UUID",
  installation_id: "Installation ID",
  hosted_app: "Hosted app",
  workspace_key: "Workspace key",
};

function integrationListUrl(orgId: string, search: string, offset: number) {
  const params = new URLSearchParams({ limit: String(PAGE_SIZE), offset: String(offset) });
  if (search) {
    params.set("search", search);
  }
  return `/admin/api/organizations/${orgId}/integrations?${params}`;
}

function earlierPageOffset(total: number, offset: number, rowCount: number) {
  if (rowCount > 0 || offset <= 0 || total <= 0) {
    return null;
  }
  const lastOffset = Math.floor((total - 1) / PAGE_SIZE) * PAGE_SIZE;
  if (lastOffset >= offset) {
    return null;
  }
  return lastOffset;
}

function integrationPage(body: AdminIntegrationsResponse, offset: number) {
  const rows = body.items ?? [];
  return { rows, total: body.total, retryOffset: earlierPageOffset(body.total, offset, rows.length) };
}

const STATE_CLASSES: Record<string, string> = {
  ready: "bg-green-50 text-green-700 dark:bg-green-900/30 dark:text-green-300",
  pending: "bg-amber-50 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300",
  error: "bg-red-50 text-red-700 dark:bg-red-900/30 dark:text-red-300",
};

export function OrgIntegrationsTable({ orgId }: { orgId: string }) {
  const [items, setItems] = useState<AdminIntegration[]>([]);
  const [total, setTotal] = useState(0);
  const [offset, setOffset] = useState(0);
  const [search, setSearch] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(false);
  const requestId = useRef(0);

  const fetchIntegrations = useCallback(
    async (nextSearch: string, nextOffset: number) => {
      const id = ++requestId.current;
      const showFailure = () => {
        setError(true);
        setLoading(false);
      };
      const showPage = (rows: AdminIntegration[], nextTotal: number) => {
        setItems(rows);
        setTotal(nextTotal);
        setError(false);
        setLoading(false);
      };
      setLoading(true);
      setError(false);
      try {
        const response = await fetch(integrationListUrl(orgId, nextSearch, nextOffset), { credentials: "include" });
        if (id !== requestId.current) {
          return;
        }
        if (!response.ok) {
          showFailure();
          return;
        }
        const body = (await response.json()) as AdminIntegrationsResponse;
        if (id !== requestId.current) {
          return;
        }
        const page = integrationPage(body, nextOffset);
        if (page.retryOffset !== null) {
          setOffset(page.retryOffset);
          void fetchIntegrations(nextSearch, page.retryOffset);
          return;
        }
        showPage(page.rows, page.total);
      } catch {
        if (id !== requestId.current) {
          return;
        }
        showFailure();
      }
    },
    [orgId],
  );

  useEffect(() => {
    const timer = setTimeout(() => {
      setOffset(0);
      void fetchIntegrations(search, 0);
    }, 200);
    return () => clearTimeout(timer);
  }, [search, fetchIntegrations]);

  return (
    <div className="mb-8">
      <div className="mb-3 flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Plug size={16} className="text-gray-600 dark:text-gray-400" />
          <Heading level={2} className="text-base text-gray-800 dark:text-gray-100">
            Connections ({total})
          </Heading>
        </div>
        <div className="relative w-56">
          <Label htmlFor="admin-connection-search" className="sr-only">
            Search connections
          </Label>
          <Search size={14} className="absolute top-1/2 left-3 -translate-y-1/2 text-gray-400 dark:text-gray-500" />
          <Input
            id="admin-connection-search"
            type="text"
            placeholder="Search connections..."
            value={search}
            onChange={(event) => setSearch(event.target.value)}
            className="pl-9"
          />
        </div>
      </div>
      <OrgIntegrationsBody
        isLoading={loading}
        isError={error}
        items={items}
        hasSearch={search !== ""}
        onClearSearch={() => {
          setSearch("");
          setLoading(true);
        }}
        onReload={() => {
          void fetchIntegrations(search, offset);
        }}
      />
      {!loading && items.length > 0 ? (
        <AdminPagination
          offset={offset}
          total={total}
          pageSize={PAGE_SIZE}
          onPageChange={(nextOffset) => {
            setOffset(nextOffset);
            void fetchIntegrations(search, nextOffset);
          }}
        />
      ) : null}
    </div>
  );
}

function OrgIntegrationsBody({
  isLoading,
  isError,
  items,
  hasSearch,
  onClearSearch,
  onReload,
}: {
  isLoading: boolean;
  isError: boolean;
  items: AdminIntegration[];
  hasSearch: boolean;
  onClearSearch: () => void;
  onReload: () => void;
}) {
  if (isLoading) {
    return <Text className="text-sm text-gray-500 dark:text-gray-400">Loading...</Text>;
  }
  if (isError) {
    return <Text className="text-sm text-red-600 dark:text-red-400">{LOAD_ERROR}</Text>;
  }
  if (items.length === 0) {
    return <OrgIntegrationsEmpty hasSearch={hasSearch} onClearSearch={onClearSearch} onReload={onReload} />;
  }

  return (
    <div className="overflow-x-auto rounded-md bg-white shadow-sm outline outline-slate-950/10 dark:bg-gray-900 dark:outline-gray-700/70">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-slate-100 dark:border-gray-700/70">
            <th className="px-4 py-2.5 text-left font-medium text-gray-500 dark:text-gray-400">Connection</th>
            <th className="px-4 py-2.5 text-left font-medium text-gray-500 dark:text-gray-400">Status</th>
            <th className="px-4 py-2.5 text-left font-medium text-gray-500 dark:text-gray-400">Details</th>
            <th className="px-4 py-2.5 text-left font-medium text-gray-500 dark:text-gray-400">Updated</th>
          </tr>
        </thead>
        <tbody>
          {items.map((item) => (
            <tr key={item.id} className="border-b border-slate-50 align-top last:border-0 dark:border-gray-800/70">
              <td className="px-4 py-2.5">
                <div className="font-medium break-all text-gray-800 dark:text-gray-100">{item.installation_name}</div>
                <div className="text-xs text-gray-500 dark:text-gray-400">{item.app_name}</div>
                <div className="font-mono text-xs break-all text-gray-400 dark:text-gray-500">{item.id}</div>
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

function OrgIntegrationsEmpty({
  hasSearch,
  onClearSearch,
  onReload,
}: {
  hasSearch: boolean;
  onClearSearch: () => void;
  onReload: () => void;
}) {
  if (hasSearch) {
    return (
      <div className="space-y-2">
        <Text className="text-sm text-gray-500 dark:text-gray-400">No connections match this search.</Text>
        <Text className="text-sm text-gray-500 dark:text-gray-400">Try a different name, or clear the search.</Text>
        <Button type="button" variant="outline" size="sm" onClick={onClearSearch}>
          Clear search
        </Button>
      </div>
    );
  }

  return (
    <div className="space-y-2">
      <Text className="text-sm text-gray-500 dark:text-gray-400">This organization has no connections.</Text>
      <Text className="text-sm text-gray-500 dark:text-gray-400">
        Members add connections in the organization. Reload this list after they add one.
      </Text>
      <Button type="button" variant="outline" size="sm" onClick={onReload}>
        Reload connections
      </Button>
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
