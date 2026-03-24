import { Text } from "@/components/Text/text";
import { Button } from "@/components/ui/button";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/ui/dropdownMenu";
import { useAccount } from "@/contexts/AccountContext";
import { showErrorToast, showSuccessToast } from "@/utils/toast";
import { ChevronDown, Eye } from "lucide-react";
import React, { useCallback, useEffect, useState } from "react";
import { AccountRow } from "./AccountRow";
import AdminPagination from "./AdminPagination";
import AdminSearchHeader from "./AdminSearchHeader";
import ConfirmAdminDialog from "./ConfirmAdminDialog";

interface Membership {
  organization_id: string;
  organization_name: string;
  user_id: string;
}
interface AdminAccount {
  id: string;
  name: string;
  email: string;
  installation_admin: boolean;
  memberships: Membership[];
}

const PAGE_SIZE = 50;

function ImpersonateBtn({ acc, onImpersonate }: { acc: AdminAccount; onImpersonate: (o: string, u: string) => void }) {
  if (acc.memberships.length === 0) return <Text className="text-xs text-gray-400">No orgs</Text>;
  if (acc.memberships.length === 1) {
    const m = acc.memberships[0];
    return (
      <Button variant="outline" size="sm" onClick={() => onImpersonate(m.organization_id, m.user_id)}>
        <span className="flex items-center gap-1">
          <Eye size={14} />
          Impersonate
        </span>
      </Button>
    );
  }
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="outline" size="sm">
          <span className="flex items-center gap-1">
            <Eye size={14} />
            Impersonate
            <ChevronDown size={12} />
          </span>
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        {acc.memberships.map((m) => (
          <DropdownMenuItem key={m.organization_id} onClick={() => onImpersonate(m.organization_id, m.user_id)}>
            {m.organization_name}
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
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

  const fetchAccounts = useCallback(async (s: string, o: number) => {
    setLoading(true);
    try {
      const params = new URLSearchParams({ limit: String(PAGE_SIZE), offset: String(o) });
      if (s) params.set("search", s);
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
      fetchAccounts(search, 0);
    }, 200);
    return () => clearTimeout(t);
  }, [search, fetchAccounts]);

  const executeToggle = async (acc: AdminAccount) => {
    setConfirmTarget(null);
    setToggling(acc.id);
    const action = acc.installation_admin ? "demote" : "promote";
    try {
      const res = await fetch(`/admin/api/accounts/${acc.id}/${action}`, { method: "POST", credentials: "include" });
      if (!res.ok) {
        showErrorToast((await res.text()) || `Failed to ${action}`);
        return;
      }
      showSuccessToast(acc.installation_admin ? `${acc.name} removed as admin` : `${acc.name} promoted to admin`);
      fetchAccounts(search, offset);
    } catch {
      showErrorToast(`Failed to ${action}`);
    } finally {
      setToggling(null);
    }
  };

  const handleImpersonate = async (orgId: string, userId: string) => {
    try {
      const res = await fetch("/admin/api/impersonate/start", {
        method: "POST",
        credentials: "include",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ organization_id: orgId, user_id: userId }),
      });
      if (!res.ok) {
        showErrorToast((await res.text()) || "Failed");
        return;
      }
      showSuccessToast("Impersonation started");
      window.location.href = (await res.json()).redirect_url;
    } catch {
      showErrorToast("Failed to start impersonation");
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
        onConfirm={() => confirmTarget && executeToggle(confirmTarget)}
        accountName={confirmTarget?.name ?? ""}
        accountEmail={confirmTarget?.email ?? ""}
        isPromoting={confirmTarget != null && !confirmTarget.installation_admin}
      />
      <AdminSearchHeader
        title="Accounts"
        subtitle={`${total} account${total !== 1 ? "s" : ""} across this installation`}
        search={search}
        onSearchChange={setSearch}
        placeholder="Search by name or email..."
      />
      {accounts.length === 0 ? (
        <div className="text-center py-12">
          <Text className="text-gray-500">{search ? "No accounts match your search." : "No accounts found."}</Text>
        </div>
      ) : (
        <>
          <div className="bg-white rounded-md shadow-sm outline outline-slate-950/10 overflow-hidden">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-slate-100">
                  <th className="text-left px-4 py-2.5 text-gray-500 font-medium">Name</th>
                  <th className="text-left px-4 py-2.5 text-gray-500 font-medium">Email</th>
                  <th className="text-left px-4 py-2.5 text-gray-500 font-medium">Access</th>
                  <th className="text-right px-4 py-2.5 text-gray-500 font-medium">Actions</th>
                </tr>
              </thead>
              <tbody>
                {accounts.map((acc) => (
                  <AccountRow
                    key={acc.id}
                    acc={acc}
                    isSelf={acc.id === currentAccount?.id}
                    toggling={toggling === acc.id}
                    onPromoteDemote={() => setConfirmTarget(acc)}
                    onImpersonate={handleImpersonate}
                    impersonateButton={<ImpersonateBtn acc={acc} onImpersonate={handleImpersonate} />}
                  />
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
              fetchAccounts(search, o);
            }}
          />
        </>
      )}
    </div>
  );
};

export default AccountsList;
