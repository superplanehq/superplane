import { Text } from "@/components/Text/text";
import { formatUsdCents } from "@/pages/factories/lib/workOrderUsage";
import { Building, CircleCheck, ClipboardList, Palette, User } from "lucide-react";
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
  remaining_credit_cents?: number;
  created_at?: string;
}

type SortField =
  | "canvas_count"
  | "created_at"
  | "done_task_count"
  | "member_count"
  | "name"
  | "remaining_credit_cents"
  | "task_count";

const PAGE_SIZE = 50;

interface OrganizationsTableProps {
  organizations: AdminOrganization[];
  sortBy: SortField;
  sortDirection: SortDirection;
  onSort: (field: SortField) => void;
}

function organizationPath(orgId: string): string {
  return `/admin/organizations/${orgId}`;
}

function openOrganization(path: string, event: React.MouseEvent, navigate: (path: string) => void) {
  if (event.metaKey || event.ctrlKey || event.shiftKey) {
    window.open(path, "_blank", "noopener,noreferrer");
    return;
  }
  navigate(path);
}

function CountCell({ icon, value }: { icon: React.ReactNode; value: number }) {
  return (
    <td className="px-4 py-2.5">
      <span className="inline-flex items-center gap-1.5 text-gray-500 dark:text-gray-400">
        {icon}
        {value}
      </span>
    </td>
  );
}

function OrganizationRow({ organization }: { organization: AdminOrganization }) {
  const navigate = useNavigate();
  const path = organizationPath(organization.id);

  return (
    <tr
      className="relative cursor-pointer border-b border-slate-50 last:border-0 hover:bg-slate-50 transition-colors dark:border-gray-800/70 dark:hover:bg-gray-800/50"
      onClick={(event) => openOrganization(path, event, navigate)}
    >
      <td className="px-4 py-2.5">
        <Link
          to={path}
          onClick={(event) => event.stopPropagation()}
          className="flex items-center gap-2 text-gray-800 hover:text-blue-600 transition-colors font-medium before:absolute before:inset-0 before:z-10 before:content-[''] dark:text-gray-100 dark:hover:text-blue-400"
        >
          <Building size={14} className="text-gray-400 shrink-0 dark:text-gray-500" />
          {organization.name || (
            <span className="text-gray-400 italic dark:text-gray-500" title={organization.id}>
              {organization.id.slice(0, 8)}...
            </span>
          )}
        </Link>
      </td>
      <CountCell icon={<Palette size={13} />} value={organization.canvas_count} />
      <CountCell icon={<ClipboardList size={13} />} value={organization.task_count} />
      <CountCell icon={<CircleCheck size={13} />} value={organization.done_task_count} />
      <CountCell icon={<User size={13} />} value={organization.member_count} />
      <td className="px-4 py-2.5 text-gray-500 whitespace-nowrap dark:text-gray-400">
        {formatUsdCents(organization.remaining_credit_cents ?? 0)}
      </td>
      <td className="px-4 py-2.5 text-gray-400 text-xs whitespace-nowrap dark:text-gray-500">
        {formatDate(organization.created_at)}
      </td>
    </tr>
  );
}

function OrganizationsTable({ organizations, sortBy, sortDirection, onSort }: OrganizationsTableProps) {
  return (
    <div className="bg-white rounded-md shadow-sm outline outline-slate-950/10 overflow-hidden dark:bg-gray-900 dark:outline-gray-700/70">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-slate-100 dark:border-gray-700/70">
            <SortableHeader
              label="Name"
              field="name"
              currentSort={sortBy}
              currentDirection={sortDirection}
              onSort={onSort}
            />
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
              label="Credits"
              field="remaining_credit_cents"
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
        </thead>
        <tbody>
          {organizations.map((organization) => (
            <OrganizationRow key={organization.id} organization={organization} />
          ))}
        </tbody>
      </table>
    </div>
  );
}

const OrganizationsList: React.FC = () => {
  const [organizations, setOrganizations] = useState<AdminOrganization[]>([]);
  const [total, setTotal] = useState(0);
  const [offset, setOffset] = useState(0);
  const [search, setSearch] = useState("");
  const [loading, setLoading] = useState(true);
  const [sortBy, setSortBy] = useState<SortField>("created_at");
  const [sortDirection, setSortDirection] = useState<SortDirection>("desc");

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
          setOrganizations(data.items);
          setTotal(data.total);
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

  useReportPageReady(!loading || organizations.length > 0);

  if (loading && organizations.length === 0) {
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
        subtitle={`${total} organization${total !== 1 ? "s" : ""} across this installation`}
        search={search}
        onSearchChange={setSearch}
        placeholder="Search by name or ID..."
      />
      {organizations.length === 0 ? (
        <div className="text-center py-12">
          <Text className="text-gray-500 dark:text-gray-400">
            {search ? "No organizations match your search." : "No organizations found."}
          </Text>
        </div>
      ) : (
        <>
          <OrganizationsTable
            organizations={organizations}
            sortBy={sortBy}
            sortDirection={sortDirection}
            onSort={handleSort}
          />
          <AdminPagination
            offset={offset}
            total={total}
            pageSize={PAGE_SIZE}
            onPageChange={(o: number) => {
              setOffset(o);
              fetchOrganizations(search, o, sortBy, sortDirection);
            }}
          />
        </>
      )}
    </div>
  );
};

export default OrganizationsList;
