import { Text } from "@/components/Text/text";
import { Building, Palette, User } from "lucide-react";
import React, { useCallback, useEffect, useState } from "react";
import { Link } from "react-router-dom";
import AdminPagination from "./AdminPagination";
import AdminSearchHeader from "./AdminSearchHeader";
import { formatDate } from "./formatDate";
import { SortableHeader, type SortDirection } from "./SortableHeader";

interface AdminOrganization {
  id: string;
  name: string;
  description: string;
  canvas_count: number;
  member_count: number;
  created_at?: string;
}

type SortField = "created_at" | "name";

const PAGE_SIZE = 50;

interface OrganizationsTableProps {
  organizations: AdminOrganization[];
  sortBy: SortField;
  sortDirection: SortDirection;
  onSort: (field: SortField) => void;
}

function OrganizationsTable({ organizations, sortBy, sortDirection, onSort }: OrganizationsTableProps) {
  return (
    <div className="bg-white rounded-md shadow-sm outline outline-slate-950/10 overflow-hidden">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-slate-100">
            <SortableHeader
              label="Name"
              field="name"
              currentSort={sortBy}
              currentDirection={sortDirection}
              onSort={onSort}
            />
            <th className="text-left px-4 py-2.5 text-gray-500 font-medium">Description</th>
            <th className="text-left px-4 py-2.5 text-gray-500 font-medium">Canvases</th>
            <th className="text-left px-4 py-2.5 text-gray-500 font-medium">Members</th>
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
          {organizations.map((org) => (
            <tr key={org.id} className="border-b border-slate-50 last:border-0 hover:bg-slate-50 transition-colors">
              <td className="px-4 py-2.5">
                <Link
                  to={`/admin/organizations/${org.id}`}
                  className="flex items-center gap-2 text-gray-800 hover:text-blue-600 transition-colors font-medium"
                >
                  <Building size={14} className="text-gray-400 shrink-0" />
                  {org.name || (
                    <span className="text-gray-400 italic" title={org.id}>
                      {org.id.slice(0, 8)}...
                    </span>
                  )}
                </Link>
              </td>
              <td className="px-4 py-2.5 text-gray-500 max-w-xs truncate">
                {org.description || <span className="text-gray-300">—</span>}
              </td>
              <td className="px-4 py-2.5">
                <span className="inline-flex items-center gap-1.5 text-gray-500">
                  <Palette size={13} />
                  {org.canvas_count}
                </span>
              </td>
              <td className="px-4 py-2.5">
                <span className="inline-flex items-center gap-1.5 text-gray-500">
                  <User size={13} />
                  {org.member_count}
                </span>
              </td>
              <td className="px-4 py-2.5 text-gray-400 text-xs whitespace-nowrap">{formatDate(org.created_at)}</td>
            </tr>
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

  if (loading && organizations.length === 0) {
    return (
      <div className="flex flex-col items-center space-y-4 py-12">
        <div className="animate-spin rounded-full h-8 w-8 border-b border-gray-500"></div>
        <Text className="text-gray-500">Loading organizations...</Text>
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
        placeholder="Search organizations..."
      />
      {organizations.length === 0 ? (
        <div className="text-center py-12">
          <Text className="text-gray-500">
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
