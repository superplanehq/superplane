import { Heading } from "@/components/Heading/heading";
import { Text } from "@/components/Text/text";
import { Button } from "@/components/ui/button";
import { Building, CircleCheck, ClipboardList, Palette, Pin, PinOff, User } from "lucide-react";
import React from "react";
import { useReportPageReady } from "@/hooks/useReportPageReady";
import { Link, useNavigate } from "react-router";
import AdminPagination from "./AdminPagination";
import AdminSearchHeader from "./AdminSearchHeader";
import { formatDate } from "./formatDate";
import { SortableHeader, type SortDirection } from "./SortableHeader";
import {
  ORGANIZATION_PAGE_SIZE,
  useOrganizationsList,
  type AdminOrganization,
  type OrganizationSortField,
} from "./useOrganizationsList";

type SortField = OrganizationSortField;

const PAGE_SIZE = ORGANIZATION_PAGE_SIZE;

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
          <Heading
            id="pinned-organizations-heading"
            level={2}
            className="mb-2 text-base text-gray-800 dark:text-gray-100"
          >
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
  const list = useOrganizationsList();
  const hasOrganizations = list.organizations.length > 0 || list.pinnedOrganizations.length > 0;
  useReportPageReady(!list.loading || hasOrganizations);

  if (list.loading && !hasOrganizations) {
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
        subtitle={`${list.matchTotal} organization${list.matchTotal !== 1 ? "s" : ""} across this installation`}
        search={list.search}
        onSearchChange={list.setSearch}
        placeholder="Search by name or ID..."
      />
      {list.pinError ? <Text className="mb-3 text-sm text-red-600 dark:text-red-400">{list.pinError}</Text> : null}
      {!hasOrganizations ? (
        <div className="text-center py-12">
          <Text className="text-gray-500 dark:text-gray-400">
            {list.search ? "No organizations match your search." : "No organizations found."}
          </Text>
        </div>
      ) : (
        <OrganizationTables
          organizations={list.organizations}
          pinnedOrganizations={list.pinnedOrganizations}
          pendingPinID={list.pendingPinID}
          sortBy={list.sortBy}
          sortDirection={list.sortDirection}
          onSort={list.handleSort}
          onTogglePin={list.togglePin}
          offset={list.offset}
          total={list.total}
          onPageChange={list.changePage}
        />
      )}
    </div>
  );
};

export default OrganizationsList;
