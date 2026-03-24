import { Text } from "@/components/Text/text";
import { Building, Palette, User } from "lucide-react";
import React, { useCallback, useEffect, useState } from "react";
import { Link } from "react-router-dom";
import AdminPagination from "./AdminPagination";
import AdminSearchHeader from "./AdminSearchHeader";

interface AdminOrganization {
  id: string;
  name: string;
  description: string;
  canvas_count: number;
  member_count: number;
}

const PAGE_SIZE = 50;

function OrganizationCard({ org }: { org: AdminOrganization }) {
  return (
    <Link
      to={`/admin/organizations/${org.id}`}
      className="bg-white rounded-md shadow-sm p-5 outline outline-slate-950/10 hover:outline-slate-950/20 hover:shadow-md transition-all"
    >
      <div className="flex flex-col h-full justify-between min-h-[120px]">
        <div>
          <div className="flex items-center gap-2 mb-1 text-gray-800">
            <Building size={16} />
            <h4 className="text-base font-medium truncate">{org.name}</h4>
          </div>
          {org.description && <Text className="text-xs text-gray-400 line-clamp-2">{org.description}</Text>}
        </div>
        <div className="mt-3 text-sm font-medium text-gray-500 flex gap-4">
          <div className="flex items-center gap-1.5">
            <Palette size={14} />
            {org.canvas_count}
          </div>
          <div className="flex items-center gap-1.5">
            <User size={14} />
            {org.member_count}
          </div>
        </div>
      </div>
    </Link>
  );
}

const OrganizationsList: React.FC = () => {
  const [organizations, setOrganizations] = useState<AdminOrganization[]>([]);
  const [total, setTotal] = useState(0);
  const [offset, setOffset] = useState(0);
  const [search, setSearch] = useState("");
  const [loading, setLoading] = useState(true);

  const fetchOrganizations = useCallback(async (searchTerm: string, pageOffset: number) => {
    setLoading(true);
    try {
      const params = new URLSearchParams({ limit: String(PAGE_SIZE), offset: String(pageOffset) });
      if (searchTerm) params.set("search", searchTerm);
      const response = await fetch(`/admin/api/organizations?${params}`, { credentials: "include" });
      if (response.ok) {
        const data = await response.json();
        setOrganizations(data.items);
        setTotal(data.total);
      }
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    const timeout = setTimeout(() => {
      setOffset(0);
      fetchOrganizations(search, 0);
    }, 200);
    return () => clearTimeout(timeout);
  }, [search, fetchOrganizations]);

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
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
            {organizations.map((org) => (
              <OrganizationCard key={org.id} org={org} />
            ))}
          </div>
          <AdminPagination
            offset={offset}
            total={total}
            pageSize={PAGE_SIZE}
            onPageChange={(o) => {
              setOffset(o);
              fetchOrganizations(search, o);
            }}
          />
        </>
      )}
    </div>
  );
};

export default OrganizationsList;
