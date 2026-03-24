import { Text } from "@/components/Text/text";
import { Heading } from "@/components/Heading/heading";
import { Input } from "@/components/Input/input";
import { Building, Palette, User, Search } from "lucide-react";
import React, { useCallback, useEffect, useState } from "react";
import { Link } from "react-router-dom";

interface AdminOrganization {
  id: string;
  name: string;
  description: string;
  canvas_count: number;
  member_count: number;
}

interface PaginatedResponse<T> {
  items: T[];
  total: number;
  limit: number;
  offset: number;
}

const PAGE_SIZE = 50;

const OrganizationsList: React.FC = () => {
  const [organizations, setOrganizations] = useState<AdminOrganization[]>([]);
  const [total, setTotal] = useState(0);
  const [offset, setOffset] = useState(0);
  const [search, setSearch] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchOrganizations = useCallback(async (searchTerm: string, pageOffset: number) => {
    setLoading(true);
    setError(null);

    try {
      const params = new URLSearchParams({
        limit: String(PAGE_SIZE),
        offset: String(pageOffset),
      });

      if (searchTerm) {
        params.set("search", searchTerm);
      }

      const response = await fetch(`/admin/api/organizations?${params}`, {
        credentials: "include",
      });

      if (!response.ok) {
        setError("Failed to load organizations");
        return;
      }

      const data: PaginatedResponse<AdminOrganization> = await response.json();
      setOrganizations(data.items);
      setTotal(data.total);
    } catch {
      setError("Failed to load organizations");
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

  const handlePageChange = (newOffset: number) => {
    setOffset(newOffset);
    fetchOrganizations(search, newOffset);
  };

  const totalPages = Math.ceil(total / PAGE_SIZE);
  const currentPage = Math.floor(offset / PAGE_SIZE) + 1;

  return (
    <div>
      <div className="flex items-center justify-between mb-4">
        <div>
          <Heading className="text-gray-800 mb-0.5">All Organizations</Heading>
          <Text className="text-gray-500 text-sm">
            {total} organization{total !== 1 ? "s" : ""} across this installation
          </Text>
        </div>

        <div className="relative w-72">
          <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
          <input
            type="text"
            placeholder="Search organizations..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="w-full pl-9 pr-3 py-1.5 text-sm border border-slate-200 rounded-md bg-white focus:outline-none focus:ring-1 focus:ring-blue-500 focus:border-blue-500"
          />
        </div>
      </div>

      {error && (
        <div className="p-4 rounded-md bg-red-50 border border-red-200 mb-4">
          <Text className="text-red-700 text-sm">{error}</Text>
        </div>
      )}

      {loading && organizations.length === 0 ? (
        <div className="flex flex-col items-center space-y-4 py-12">
          <div className="animate-spin rounded-full h-8 w-8 border-b border-gray-500"></div>
          <Text className="text-gray-500">Loading organizations...</Text>
        </div>
      ) : organizations.length === 0 ? (
        <div className="text-center py-12">
          <Text className="text-gray-500">
            {search ? "No organizations match your search." : "No organizations found."}
          </Text>
        </div>
      ) : (
        <>
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
            {organizations.map((org) => (
              <Link
                key={org.id}
                to={`/admin/organizations/${org.id}`}
                className="bg-white rounded-md shadow-sm p-5 outline outline-slate-950/10 hover:outline-slate-950/20 hover:shadow-md transition-all"
              >
                <div className="flex flex-col h-full justify-between min-h-[120px]">
                  <div>
                    <div className="flex items-center gap-2 mb-1 text-gray-800">
                      <Building size={16} />
                      <h4 className="text-base font-medium truncate">{org.name}</h4>
                    </div>
                    {org.description && (
                      <Text className="text-xs text-gray-400 line-clamp-2">{org.description}</Text>
                    )}
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
            ))}
          </div>

          {totalPages > 1 && (
            <div className="flex items-center justify-between mt-6 text-sm text-gray-500">
              <Text>
                Showing {offset + 1}–{Math.min(offset + PAGE_SIZE, total)} of {total}
              </Text>
              <div className="flex gap-2">
                <button
                  onClick={() => handlePageChange(offset - PAGE_SIZE)}
                  disabled={offset === 0}
                  className="px-3 py-1 rounded border border-slate-200 bg-white hover:bg-slate-50 disabled:opacity-40 disabled:cursor-not-allowed"
                >
                  Previous
                </button>
                <button
                  onClick={() => handlePageChange(offset + PAGE_SIZE)}
                  disabled={currentPage >= totalPages}
                  className="px-3 py-1 rounded border border-slate-200 bg-white hover:bg-slate-50 disabled:opacity-40 disabled:cursor-not-allowed"
                >
                  Next
                </button>
              </div>
            </div>
          )}
        </>
      )}
    </div>
  );
};

export default OrganizationsList;
