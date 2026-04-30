import { Text } from "@/components/Text/text";
import { Button } from "@/components/ui/button";
import { useAccount } from "@/contexts/AccountContext";
import { showErrorToast } from "@/lib/toast";
import { Eye } from "lucide-react";
import React, { useCallback, useEffect, useState } from "react";
import { AccountRow } from "./AccountRow";
import AdminPagination from "./AdminPagination";
import AdminSearchHeader from "./AdminSearchHeader";
import ConfirmAdminDialog from "./ConfirmAdminDialog";
import { SortableHeader, type SortDirection } from "./SortableHeader";
import { startImpersonation, toggleAdmin } from "./useAccountActions";

interface AdminAccount {
  id: string;
  name: string;
  email: string;
  installation_admin: boolean;
  created_at?: string;
}

type SortField = "created_at" | "name" | "email";

const PAGE_SIZE = 50;

interface AccountsTableProps {
  accounts: AdminAccount[];
  currentAccountId?: string;
  togglingAccountId: string | null;
  sortBy: SortField;
  sortDirection: SortDirection;
  onSort: (field: SortField) => void;
  onPromoteDemote: (account: AdminAccount) => void;
}

function AccountsTable({
  accounts,
  currentAccountId,
  togglingAccountId,
  sortBy,
  sortDirection,
  onSort,
  onPromoteDemote,
}: AccountsTableProps) {
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
            <SortableHeader
              label="Email"
              field="email"
              currentSort={sortBy}
              currentDirection={sortDirection}
              onSort={onSort}
            />
            <th className="text-left px-4 py-2.5 text-gray-500 font-medium">Access</th>
            <SortableHeader
              label="Created"
              field="created_at"
              currentSort={sortBy}
              currentDirection={sortDirection}
              onSort={onSort}
            />
            <th className="text-right px-4 py-2.5 text-gray-500 font-medium">Actions</th>
          </tr>
        </thead>
        <tbody>
          {accounts.map((acc) => (
            <AccountRow
              key={acc.id}
              acc={acc}
              isSelf={acc.id === currentAccountId}
              toggling={togglingAccountId === acc.id}
              onPromoteDemote={() => onPromoteDemote(acc)}
              impersonateButton={
                <Button variant="outline" size="sm" onClick={() => startImpersonation(acc.id)}>
                  <span className="flex items-center gap-1">
                    <Eye size={14} />
                    Impersonate
                  </span>
                </Button>
              }
            />
          ))}
        </tbody>
      </table>
    </div>
  );
}

const AccountsList: React.FC = () => {
  const { account: currentAccount } = useAccount();
  const [accounts, setAccounts] = useState<AdminAccount[]>([]);
  const [total, setTotal] = useState(0);
  const [offset, setOffset] = useState(0);
  const [search, setSearch] = useState("");
  const [loading, setLoading] = useState(true);
  const [toggling, setToggling] = useState<string | null>(null);
  const [confirmTarget, setConfirmTarget] = useState<AdminAccount | null>(null);
  const [sortBy, setSortBy] = useState<SortField>("created_at");
  const [sortDirection, setSortDirection] = useState<SortDirection>("desc");

  const fetchAccounts = useCallback(async (s: string, o: number, sort: SortField, direction: SortDirection) => {
    setLoading(true);
    try {
      const params = new URLSearchParams({ limit: String(PAGE_SIZE), offset: String(o) });
      if (s) params.set("search", s);
      params.set("sort_by", sort);
      params.set("sort_direction", direction);
      const res = await fetch(`/admin/api/accounts?${params}`, { credentials: "include" });
      if (res.ok) {
        const d = await res.json();
        setAccounts(d.items);
        setTotal(d.total);
      }
    } catch {
      showErrorToast("Failed to load accounts");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    const t = setTimeout(() => {
      setOffset(0);
      fetchAccounts(search, 0, sortBy, sortDirection);
    }, 200);
    return () => clearTimeout(t);
  }, [search, sortBy, sortDirection, fetchAccounts]);

  const onToggle = async (acc: AdminAccount) => {
    setConfirmTarget(null);
    setToggling(acc.id);
    await toggleAdmin(acc, () => fetchAccounts(search, offset, sortBy, sortDirection));
    setToggling(null);
  };

  const handleSort = (field: SortField) => {
    if (field === sortBy) {
      setSortDirection((prev) => (prev === "asc" ? "desc" : "asc"));
    } else {
      setSortBy(field);
      setSortDirection(field === "name" || field === "email" ? "asc" : "desc");
    }
  };

  if (loading && accounts.length === 0)
    return (
      <div className="flex flex-col items-center space-y-4 py-12">
        <div className="animate-spin rounded-full h-8 w-8 border-b border-gray-500"></div>
        <Text className="text-gray-500">Loading accounts...</Text>
      </div>
    );

  return (
    <div>
      <ConfirmAdminDialog
        open={confirmTarget !== null}
        onClose={() => setConfirmTarget(null)}
        onConfirm={() => confirmTarget && onToggle(confirmTarget)}
        accountName={confirmTarget?.name ?? ""}
        accountEmail={confirmTarget?.email ?? ""}
        isPromoting={confirmTarget != null && !confirmTarget.installation_admin}
      />
      <AdminSearchHeader
        title="Accounts"
        subtitle={`${total} account${total !== 1 ? "s" : ""}`}
        search={search}
        onSearchChange={setSearch}
        placeholder="Search by name or email..."
      />
      {accounts.length === 0 ? (
        <div className="text-center py-12">
          <Text className="text-gray-500">{search ? "No accounts match." : "No accounts found."}</Text>
        </div>
      ) : (
        <>
          <AccountsTable
            accounts={accounts}
            currentAccountId={currentAccount?.id}
            togglingAccountId={toggling}
            sortBy={sortBy}
            sortDirection={sortDirection}
            onSort={handleSort}
            onPromoteDemote={setConfirmTarget}
          />
          <AdminPagination
            offset={offset}
            total={total}
            pageSize={PAGE_SIZE}
            onPageChange={(o: number) => {
              setOffset(o);
              fetchAccounts(search, o, sortBy, sortDirection);
            }}
          />
        </>
      )}
    </div>
  );
};

export default AccountsList;
