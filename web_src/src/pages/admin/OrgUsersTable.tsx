import { Text } from "@/components/Text/text";
import { Heading } from "@/components/Heading/heading";
import { Button } from "@/components/ui/button";
import { useAccount } from "@/contexts/AccountContext";
import { Search, User as UserIcon } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import AdminPagination from "./AdminPagination";
import { startImpersonation } from "./useAccountActions";

interface OrgUser {
  id: string;
  name: string;
  email: string | null;
  account_id: string | null;
}
interface PaginatedResponse<T> {
  items: T[];
  total: number;
}

const PAGE_SIZE = 50;

function UserRow({ user }: { user: OrgUser }) {
  const { account } = useAccount();
  const isSelf = user.account_id === account?.id;

  return (
    <tr className="border-b border-slate-50 last:border-0">
      <td className="px-4 py-2.5 text-gray-800">{user.name}</td>
      <td className="px-4 py-2.5 text-gray-500">{user.email || "—"}</td>
      <td className="px-4 py-2.5 text-right">
        {isSelf ? (
          <Text className="text-xs text-gray-400">You</Text>
        ) : user.account_id ? (
          <Button variant="outline" size="sm" onClick={() => startImpersonation(user.account_id!)}>
            Impersonate
          </Button>
        ) : null}
      </td>
    </tr>
  );
}

export function OrgUsersTable({ orgId }: { orgId: string }) {
  const [users, setUsers] = useState<OrgUser[]>([]);
  const [total, setTotal] = useState(0);
  const [offset, setOffset] = useState(0);
  const [search, setSearch] = useState("");

  const fetchUsers = useCallback(
    async (s: string, o: number) => {
      const params = new URLSearchParams({ limit: String(PAGE_SIZE), offset: String(o) });
      if (s) params.set("search", s);
      const res = await fetch(`/admin/api/organizations/${orgId}/users?${params}`, { credentials: "include" });
      if (res.ok) {
        const d: PaginatedResponse<OrgUser> = await res.json();
        setUsers(d.items);
        setTotal(d.total);
      }
    },
    [orgId],
  );

  useEffect(() => {
    const t = setTimeout(() => {
      setOffset(0);
      fetchUsers(search, 0);
    }, 200);
    return () => clearTimeout(t);
  }, [search, fetchUsers]);

  return (
    <div className="mb-8">
      <div className="flex items-center justify-between mb-3">
        <div className="flex items-center gap-2">
          <UserIcon size={16} className="text-gray-600" />
          <Heading level={2} className="text-gray-800 text-base">
            Users ({total})
          </Heading>
        </div>
        <div className="relative w-56">
          <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
          <input
            type="text"
            placeholder="Search users..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="w-full pl-9 pr-3 py-1.5 text-sm border border-slate-200 rounded-md bg-white focus:outline-none focus:ring-1 focus:ring-blue-500"
          />
        </div>
      </div>
      {users.length === 0 ? (
        <Text className="text-gray-500 text-sm">
          {search ? "No users match your search." : "No users in this organization."}
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
                {users.map((u) => (
                  <UserRow key={u.id} user={u} />
                ))}
              </tbody>
            </table>
          </div>
          <AdminPagination
            offset={offset}
            total={total}
            pageSize={PAGE_SIZE}
            onPageChange={(o) => {
              setOffset(o);
              fetchUsers(search, o);
            }}
          />
        </>
      )}
    </div>
  );
}
