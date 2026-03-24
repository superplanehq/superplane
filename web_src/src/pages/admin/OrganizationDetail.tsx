import { Text } from "@/components/Text/text";
import { Heading } from "@/components/Heading/heading";
import { Button } from "@/components/ui/button";
import { useAccount } from "@/contexts/AccountContext";
import { showErrorToast, showSuccessToast } from "@/utils/toast";
import { ArrowLeft, Palette, Search, User as UserIcon } from "lucide-react";
import React, { useCallback, useEffect, useState } from "react";
import { Link, useParams } from "react-router-dom";

interface Canvas {
  id: string;
  name: string;
  description: string;
}

interface OrgUser {
  id: string;
  name: string;
  email: string | null;
  account_id: string | null;
}

interface PaginatedResponse<T> {
  items: T[];
  total: number;
  limit: number;
  offset: number;
}

const PAGE_SIZE = 50;

const OrganizationDetail: React.FC = () => {
  const { orgId } = useParams<{ orgId: string }>();
  const { account } = useAccount();

  const [canvases, setCanvases] = useState<Canvas[]>([]);
  const [canvasTotal, setCanvasTotal] = useState(0);
  const [canvasOffset, setCanvasOffset] = useState(0);
  const [canvasSearch, setCanvasSearch] = useState("");

  const [users, setUsers] = useState<OrgUser[]>([]);
  const [userTotal, setUserTotal] = useState(0);
  const [userOffset, setUserOffset] = useState(0);
  const [userSearch, setUserSearch] = useState("");

  const [loading, setLoading] = useState(true);
  const [impersonating, setImpersonating] = useState<string | null>(null);

  const fetchUsers = useCallback(
    async (search: string, offset: number) => {
      const params = new URLSearchParams({ limit: String(PAGE_SIZE), offset: String(offset) });
      if (search) params.set("search", search);

      const res = await fetch(`/admin/api/organizations/${orgId}/users?${params}`, { credentials: "include" });
      if (res.ok) {
        const data: PaginatedResponse<OrgUser> = await res.json();
        setUsers(data.items);
        setUserTotal(data.total);
      }
    },
    [orgId],
  );

  const fetchCanvases = useCallback(
    async (search: string, offset: number) => {
      const params = new URLSearchParams({ limit: String(PAGE_SIZE), offset: String(offset) });
      if (search) params.set("search", search);

      const res = await fetch(`/admin/api/organizations/${orgId}/canvases?${params}`, { credentials: "include" });
      if (res.ok) {
        const data: PaginatedResponse<Canvas> = await res.json();
        setCanvases(data.items);
        setCanvasTotal(data.total);
      }
    },
    [orgId],
  );

  // Initial load
  useEffect(() => {
    Promise.all([fetchUsers("", 0), fetchCanvases("", 0)]).finally(() => setLoading(false));
  }, [fetchUsers, fetchCanvases]);

  // Debounced user search
  useEffect(() => {
    const t = setTimeout(() => {
      setUserOffset(0);
      fetchUsers(userSearch, 0);
    }, 200);
    return () => clearTimeout(t);
  }, [userSearch, fetchUsers]);

  // Debounced canvas search
  useEffect(() => {
    const t = setTimeout(() => {
      setCanvasOffset(0);
      fetchCanvases(canvasSearch, 0);
    }, 200);
    return () => clearTimeout(t);
  }, [canvasSearch, fetchCanvases]);

  const handleImpersonate = async (userId: string) => {
    setImpersonating(userId);

    try {
      const response = await fetch("/admin/api/impersonate/start", {
        method: "POST",
        credentials: "include",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ organization_id: orgId, user_id: userId }),
      });

      if (!response.ok) {
        const errorText = await response.text();
        showErrorToast(errorText || "Failed to start impersonation");
        return;
      }

      const data = await response.json();
      showSuccessToast("Impersonation started");
      window.location.href = data.redirect_url;
    } catch {
      showErrorToast("Failed to start impersonation");
    } finally {
      setImpersonating(null);
    }
  };

  if (loading) {
    return (
      <div className="flex flex-col items-center space-y-4 py-12">
        <div className="animate-spin rounded-full h-8 w-8 border-b border-gray-500"></div>
        <Text className="text-gray-500">Loading...</Text>
      </div>
    );
  }

  const userPages = Math.ceil(userTotal / PAGE_SIZE);
  const canvasPages = Math.ceil(canvasTotal / PAGE_SIZE);

  return (
    <div>
      <Link to="/admin" className="inline-flex items-center gap-1.5 text-sm text-gray-500 hover:text-gray-700 mb-4">
        <ArrowLeft size={14} />
        All organizations
      </Link>

      {/* Users section */}
      <div className="mb-8">
        <div className="flex items-center justify-between mb-3">
          <div className="flex items-center gap-2">
            <UserIcon size={16} className="text-gray-600" />
            <Heading level={2} className="text-gray-800 text-base">
              Users ({userTotal})
            </Heading>
          </div>
          <div className="relative w-56">
            <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
            <input
              type="text"
              placeholder="Search users..."
              value={userSearch}
              onChange={(e) => setUserSearch(e.target.value)}
              className="w-full pl-9 pr-3 py-1.5 text-sm border border-slate-200 rounded-md bg-white focus:outline-none focus:ring-1 focus:ring-blue-500"
            />
          </div>
        </div>

        {users.length === 0 ? (
          <Text className="text-gray-500 text-sm">
            {userSearch ? "No users match your search." : "No users in this organization."}
          </Text>
        ) : (
          <>
            <div className="bg-white rounded-md shadow-sm outline outline-slate-950/10 overflow-hidden">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-slate-100">
                    <th className="text-left px-4 py-2.5 text-gray-500 font-medium">Name</th>
                    <th className="text-left px-4 py-2.5 text-gray-500 font-medium">Email</th>
                    <th className="text-right px-4 py-2.5 text-gray-500 font-medium">Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {users.map((user) => (
                    <tr key={user.id} className="border-b border-slate-50 last:border-0">
                      <td className="px-4 py-2.5 text-gray-800">{user.name}</td>
                      <td className="px-4 py-2.5 text-gray-500">{user.email || "—"}</td>
                      <td className="px-4 py-2.5 text-right">
                        {user.account_id === account?.id ? (
                          <Text className="text-xs text-gray-400">You</Text>
                        ) : (
                          <Button
                            variant="outline"
                            size="sm"
                            onClick={() => handleImpersonate(user.id)}
                            disabled={impersonating === user.id}
                          >
                            {impersonating === user.id ? "Starting..." : "Impersonate"}
                          </Button>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>

            {userPages > 1 && (
              <div className="flex items-center justify-between mt-3 text-sm text-gray-500">
                <Text>
                  Showing {userOffset + 1}–{Math.min(userOffset + PAGE_SIZE, userTotal)} of {userTotal}
                </Text>
                <div className="flex gap-2">
                  <button
                    onClick={() => {
                      const o = userOffset - PAGE_SIZE;
                      setUserOffset(o);
                      fetchUsers(userSearch, o);
                    }}
                    disabled={userOffset === 0}
                    className="px-3 py-1 rounded border border-slate-200 bg-white hover:bg-slate-50 disabled:opacity-40 disabled:cursor-not-allowed text-xs"
                  >
                    Previous
                  </button>
                  <button
                    onClick={() => {
                      const o = userOffset + PAGE_SIZE;
                      setUserOffset(o);
                      fetchUsers(userSearch, o);
                    }}
                    disabled={Math.floor(userOffset / PAGE_SIZE) + 1 >= userPages}
                    className="px-3 py-1 rounded border border-slate-200 bg-white hover:bg-slate-50 disabled:opacity-40 disabled:cursor-not-allowed text-xs"
                  >
                    Next
                  </button>
                </div>
              </div>
            )}
          </>
        )}
      </div>

      {/* Canvases section */}
      <div>
        <div className="flex items-center justify-between mb-3">
          <div className="flex items-center gap-2">
            <Palette size={16} className="text-gray-600" />
            <Heading level={2} className="text-gray-800 text-base">
              Canvases ({canvasTotal})
            </Heading>
          </div>
          <div className="relative w-56">
            <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
            <input
              type="text"
              placeholder="Search canvases..."
              value={canvasSearch}
              onChange={(e) => setCanvasSearch(e.target.value)}
              className="w-full pl-9 pr-3 py-1.5 text-sm border border-slate-200 rounded-md bg-white focus:outline-none focus:ring-1 focus:ring-blue-500"
            />
          </div>
        </div>

        {canvases.length === 0 ? (
          <Text className="text-gray-500 text-sm">
            {canvasSearch ? "No canvases match your search." : "No canvases in this organization."}
          </Text>
        ) : (
          <>
            <div className="bg-white rounded-md shadow-sm outline outline-slate-950/10 overflow-hidden">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-slate-100">
                    <th className="text-left px-4 py-2.5 text-gray-500 font-medium">Name</th>
                    <th className="text-left px-4 py-2.5 text-gray-500 font-medium">Description</th>
                  </tr>
                </thead>
                <tbody>
                  {canvases.map((canvas) => (
                    <tr key={canvas.id} className="border-b border-slate-50 last:border-0">
                      <td className="px-4 py-2.5 text-gray-800 font-medium">{canvas.name}</td>
                      <td className="px-4 py-2.5 text-gray-500">{canvas.description || "—"}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>

            {canvasPages > 1 && (
              <div className="flex items-center justify-between mt-3 text-sm text-gray-500">
                <Text>
                  Showing {canvasOffset + 1}–{Math.min(canvasOffset + PAGE_SIZE, canvasTotal)} of {canvasTotal}
                </Text>
                <div className="flex gap-2">
                  <button
                    onClick={() => {
                      const o = canvasOffset - PAGE_SIZE;
                      setCanvasOffset(o);
                      fetchCanvases(canvasSearch, o);
                    }}
                    disabled={canvasOffset === 0}
                    className="px-3 py-1 rounded border border-slate-200 bg-white hover:bg-slate-50 disabled:opacity-40 disabled:cursor-not-allowed text-xs"
                  >
                    Previous
                  </button>
                  <button
                    onClick={() => {
                      const o = canvasOffset + PAGE_SIZE;
                      setCanvasOffset(o);
                      fetchCanvases(canvasSearch, o);
                    }}
                    disabled={Math.floor(canvasOffset / PAGE_SIZE) + 1 >= canvasPages}
                    className="px-3 py-1 rounded border border-slate-200 bg-white hover:bg-slate-50 disabled:opacity-40 disabled:cursor-not-allowed text-xs"
                  >
                    Next
                  </button>
                </div>
              </div>
            )}
          </>
        )}
      </div>
    </div>
  );
};

export default OrganizationDetail;
