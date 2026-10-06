import { Heading } from "@/components/Heading/heading";
import { Text } from "@/components/Text/text";
import { Button } from "@/components/ui/button";
import { Building, CircleCheck, ClipboardList, Palette, Pin, PinOff, User } from "lucide-react";
import React, { useCallback, useEffect, useState } from "react";
import { useReportPageReady } from "@/hooks/useReportPageReady";
import { Link, useNavigate } from "react-router";
import AdminPagination from "./AdminPagination";
import AdminSearchHeader from "./AdminSearchHeader";
import { formatDate } from "./formatDate";
import { SortableHeader, type SortDirection } from "./SortableHeader";

interface AdminOrganization {
  id: string;
  name: string;
  canvas_count: number;
  task_count: number;
  done_task_count: number;
  member_count: number;
  created_at?: string;
}

type SortField = "canvas_count" | "created_at" | "done_task_count" | "member_count" | "name" | "task_count";

const PAGE_SIZE = 50;

interface OrganizationsTableProps {
  organizations: AdminOrganization[];
  pinned: boolean;
  pendingPinID: string | null;
  sortBy: SortField;
  sortDirection: SortDirection;
  onSort: (field: SortField) => void;
  onTogglePin: (organization: AdminOrganization, pinned: boolean) => void;
}

function organizationPath(orgId: string): string {
  return `/admin/organizations/${orgId}`;
}

function pinActionLabel(pinned: boolean): string {
  return pinned ? "Unpin organization" : "Pin organization";
}

function OrganizationTableHeader({
  sortBy,
  sortDirection,
  onSort,
}: {
  sortBy: SortField;
  sortDirection: SortDirection;
  onSort: (field: SortField) => void;
}) {
  return (
    <tr className="border-b border-slate-100 dark:border-gray-700/70">
      <SortableHeader label="Name" field="name" currentSort={sortBy} currentDirection={sortDirection} onSort={onSort} />
      <SortableHeader
        label="Automations"
        field="canvas_count"
        currentSort={sortBy}
        currentDirection={sortDirection}
        onSort={onSort}
      />
      <SortableHeader
        label="Tasks"
        field="task_count"
        currentSort={sortBy}
        currentDirection={sortDirection}
        onSort={onSort}
      />
      <SortableHeader
        label="Done Tasks"
        field="done_task_count"
        currentSort={sortBy}
        currentDirection={sortDirection}
        onSort={onSort}
      />
      <SortableHeader
        label="Members"
        field="member_count"
        currentSort={sortBy}
        currentDirection={sortDirection}
        onSort={onSort}
      />
      <SortableHeader
        label="Created"
        field="created_at"
        currentSort={sortBy}
        currentDirection={sortDirection}
        onSort={onSort}
      />
    </tr>
  );
}

function OrganizationRow({
  organization,
  pinned,
  pending,
  onTogglePin,
}: {
  organization: AdminOrganization;
  pinned: boolean;
  pending: boolean;
  onTogglePin: (organization: AdminOrganization, pinned: boolean) => void;
}) {
  const navigate = useNavigate();

  return (
    <tr
      className="relative cursor-pointer border-b border-slate-50 last:border-0 hover:bg-slate-50 transition-colors dark:border-gray-800/70 dark:hover:bg-gray-800/50"
      onClick={(event) => {
        const path = organizationPath(organization.id);
        if (event.metaKey || event.ctrlKey || event.shiftKey) {
          window.open(path, "_blank", "noopener,noreferrer");
          return;
        }
        navigate(path);
      }}
    >
      <td className="px-4 py-2.5">
        <div className="flex items-center gap-2">
          <Link
            to={organizationPath(organization.id)}
            onClick={(event) => event.stopPropagation()}
            className="flex min-w-0 flex-1 items-center gap-2 text-gray-800 hover:text-blue-600 transition-colors font-medium before:absolute before:inset-0 before:z-10 before:content-[''] dark:text-gray-100 dark:hover:text-blue-400"
          >
            <Building size={14} className="text-gray-400 shrink-0 dark:text-gray-500" />
            {organization.name || (
              <span className="text-gray-400 italic dark:text-gray-500" title={organization.id}>
                {organization.id.slice(0, 8)}...
              </span>
            )}
          </Link>
          <Button
            type="button"
            variant="ghost"
            size="icon-xs"
            aria-label={pinActionLabel(pinned)}
            className="relative z-20 shrink-0"
            disabled={pending}
            onClick={(event) => {
              event.stopPropagation();
              onTogglePin(organization, pinned);
            }}
          >
            {pinned ? <PinOff /> : <Pin />}
          </Button>
        </div>
      </td>
      <td className="px-4 py-2.5">
        <span className="inline-flex items-center gap-1.5 text-gray-500 dark:text-gray-400">
          <Palette size={13} />
          {organization.canvas_count}
        </span>
      </td>
      <td className="px-4 py-2.5">
        <span className="inline-flex items-center gap-1.5 text-gray-500 dark:text-gray-400">
          <ClipboardList size={13} />
          {organization.task_count}
        </span>
      </td>
      <td className="px-4 py-2.5">
        <span className="inline-flex items-center gap-1.5 text-gray-500 dark:text-gray-400">
          <CircleCheck size={13} />
          {organization.done_task_count}
        </span>
      </td>
      <td className="px-4 py-2.5">
        <span className="inline-flex items-center gap-1.5 text-gray-500 dark:text-gray-400">
          <User size={13} />
          {organization.member_count}
        </span>
      </td>
      <td className="px-4 py-2.5 text-gray-400 text-xs whitespace-nowrap dark:text-gray-500">
        {formatDate(organization.created_at)}
      </td>
    </tr>
  );
}

function OrganizationsTable({
  organizations,
  pinned,
  pendingPinID,
  sortBy,
  sortDirection,
  onSort,
  onTogglePin,
}: OrganizationsTableProps) {
  return (
    <div className="bg-white rounded-md shadow-sm outline outline-slate-950/10 overflow-hidden dark:bg-gray-900 dark:outline-gray-700/70">
      <table className="w-full text-sm">
        <thead>
          <OrganizationTableHeader sortBy={sortBy} sortDirection={sortDirection} onSort={onSort} />
        </thead>
        <tbody>
          {organizations.map((organization) => (
            <OrganizationRow
              key={organization.id}
              organization={organization}
              pinned={pinned}
              pending={pendingPinID === organization.id}
              onTogglePin={onTogglePin}
            />
          ))}
        </tbody>
      </table>
    </div>
  );
}

function OrganizationTables({
  organizations,
  pinnedOrganizations,
  pendingPinID,
  sortBy,
  sortDirection,
  onSort,
  onTogglePin,
  offset,
  total,
  onPageChange,
}: {
  organizations: AdminOrganization[];
  pinnedOrganizations: AdminOrganization[];
  pendingPinID: string | null;
  sortBy: SortField;
  sortDirection: SortDirection;
  onSort: (field: SortField) => void;
  onTogglePin: (organization: AdminOrganization, pinned: boolean) => void;
  offset: number;
  total: number;
  onPageChange: (offset: number) => void;
}) {
  return (
    <>
      {pinnedOrganizations.length > 0 ? (
        <section className="mb-6" aria-labelledby="pinned-organizations-heading">
          <Heading id="pinned-organizations-heading" level={2} className="mb-2 text-base text-gray-800 dark:text-gray-100">
            Pinned
          </Heading>
          <OrganizationsTable
            organizations={pinnedOrganizations}
            pinned
            pendingPinID={pendingPinID}
            sortBy={sortBy}
            sortDirection={sortDirection}
            onSort={onSort}
            onTogglePin={onTogglePin}
          />
        </section>
      ) : null}
      {organizations.length > 0 ? (
        <OrganizationsTable
          organizations={organizations}
          pinned={false}
          pendingPinID={pendingPinID}
          sortBy={sortBy}
          sortDirection={sortDirection}
          onSort={onSort}
          onTogglePin={onTogglePin}
        />
      ) : null}
      <AdminPagination offset={offset} total={total} pageSize={PAGE_SIZE} onPageChange={onPageChange} />
    </>
  );
}

const OrganizationsList: React.FC = () => {
  const [organizations, setOrganizations] = useState<AdminOrganization[]>([]);
  const [pinnedOrganizations, setPinnedOrganizations] = useState<AdminOrganization[]>([]);
  const [total, setTotal] = useState(0);
  const [matchTotal, setMatchTotal] = useState(0);
  const [offset, setOffset] = useState(0);
  const [search, setSearch] = useState("");
  const [loading, setLoading] = useState(true);
  const [sortBy, setSortBy] = useState<SortField>("created_at");
  const [sortDirection, setSortDirection] = useState<SortDirection>("desc");
  const [pinError, setPinError] = useState<string | null>(null);
  const [pendingPinID, setPendingPinID] = useState<string | null>(null);

  const fetchOrganizations = useCallback(
    async (searchTerm: string, pageOffset: number, sort: SortField, direction: SortDirection) => {
      setLoading(true);
      try {
        const params = new URLSearchParams({ limit: String(PAGE_SIZE), offset: String(pageOffset) });
        if (searchTerm) params.set("search", searchTerm);
        params.set("sort_by", sort);
        params.set("sort_direction", direction);
        const response = await fetch(`/admin/api/organizations?${params}`, { credentials: "include" });
        if (response.ok) {
          const data = await response.json();
          setOrganizations(data.items ?? []);
          setPinnedOrganizations(data.pinned ?? []);
          setTotal(data.total ?? 0);
          setMatchTotal(typeof data.match_total === "number" ? data.match_total : (data.total ?? 0));
        }
      } finally {
        setLoading(false);
      }
    },
    [],
  );

  useEffect(() => {
    const timeout = setTimeout(() => {
      setOffset(0);
      fetchOrganizations(search, 0, sortBy, sortDirection);
    }, 200);
    return () => clearTimeout(timeout);
  }, [search, sortBy, sortDirection, fetchOrganizations]);

  const handleSort = (field: SortField) => {
    if (field === sortBy) {
      setSortDirection((prev) => (prev === "asc" ? "desc" : "asc"));
    } else {
      setSortBy(field);
      setSortDirection(field === "name" ? "asc" : "desc");
    }
  };

  const togglePin = async (organization: AdminOrganization, pinned: boolean) => {
    setPinError(null);
    setPendingPinID(organization.id);
    try {
      const response = await fetch(`/admin/api/organizations/${organization.id}/pin`, {
        method: pinned ? "DELETE" : "PUT",
        credentials: "include",
      });
      if (!response.ok) {
        setPinError(pinned ? "Could not unpin this organization." : "Could not pin this organization.");
        return;
      }
      await fetchOrganizations(search, offset, sortBy, sortDirection);
    } catch {
      setPinError(pinned ? "Could not unpin this organization." : "Could not pin this organization.");
    } finally {
      setPendingPinID(null);
    }
  };

  const hasOrganizations = organizations.length > 0 || pinnedOrganizations.length > 0;
  useReportPageReady(!loading || hasOrganizations);

  if (loading && !hasOrganizations) {
    return (
      <div className="flex flex-col items-center space-y-4 py-12">
        <div className="animate-spin rounded-full h-8 w-8 border-b border-gray-500 dark:border-gray-400"></div>
        <Text className="text-gray-500 dark:text-gray-400">Loading organizations...</Text>
      </div>
    );
  }

  return (
    <div>
      <AdminSearchHeader
        title="All Organizations"
        subtitle={`${matchTotal} organization${matchTotal !== 1 ? "s" : ""} across this installation`}
        search={search}
        onSearchChange={setSearch}
        placeholder="Search by name or ID..."
      />
      {pinError ? <Text className="mb-3 text-sm text-red-600 dark:text-red-400">{pinError}</Text> : null}
      {!hasOrganizations ? (
        <div className="text-center py-12">
          <Text className="text-gray-500 dark:text-gray-400">
            {search ? "No organizations match your search." : "No organizations found."}
          </Text>
        </div>
      ) : (
        <>
          <OrganizationTables
            organizations={organizations}
            pinnedOrganizations={pinnedOrganizations}
            pendingPinID={pendingPinID}
            sortBy={sortBy}
            sortDirection={sortDirection}
            onSort={handleSort}
            onTogglePin={togglePin}
            offset={offset}
            total={total}
            onPageChange={(pageOffset: number) => {
              setOffset(pageOffset);
              void fetchOrganizations(search, pageOffset, sortBy, sortDirection);
            }}
          />
        </>
      )}
    </div>
  );
};

export default OrganizationsList;
