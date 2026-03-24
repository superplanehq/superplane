import { Text } from "@/components/Text/text";
import { Heading } from "@/components/Heading/heading";
import { Button } from "@/components/ui/button";
import { Dialog, DialogActions, DialogDescription, DialogTitle } from "@/components/Dialog/dialog";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/ui/dropdownMenu";
import { useAccount } from "@/contexts/AccountContext";
import { showErrorToast, showSuccessToast } from "@/utils/toast";
import { AlertTriangle, ChevronDown, Eye, Search, Shield, ShieldOff } from "lucide-react";
import React, { useCallback, useEffect, useState } from "react";

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

interface PaginatedResponse<T> {
  items: T[];
  total: number;
  limit: number;
  offset: number;
}

const PAGE_SIZE = 50;

const AccountsList: React.FC = () => {
  const { account: currentAccount } = useAccount();
  const [accounts, setAccounts] = useState<AdminAccount[]>([]);
  const [total, setTotal] = useState(0);
  const [offset, setOffset] = useState(0);
  const [search, setSearch] = useState("");
  const [loading, setLoading] = useState(true);
  const [toggling, setToggling] = useState<string | null>(null);
  const [confirmTarget, setConfirmTarget] = useState<AdminAccount | null>(null);
  const [impersonating, setImpersonating] = useState<string | null>(null);

  const fetchAccounts = useCallback(async (searchTerm: string, pageOffset: number) => {
    setLoading(true);
    try {
      const params = new URLSearchParams({ limit: String(PAGE_SIZE), offset: String(pageOffset) });
      if (searchTerm) params.set("search", searchTerm);

      const response = await fetch(`/admin/api/accounts?${params}`, { credentials: "include" });
      if (response.ok) {
        const data: PaginatedResponse<AdminAccount> = await response.json();
        setAccounts(data.items);
        setTotal(data.total);
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

  const executeToggle = async (account: AdminAccount) => {
    setConfirmTarget(null);
    setToggling(account.id);
    const action = account.installation_admin ? "demote" : "promote";

    try {
      const response = await fetch(`/admin/api/accounts/${account.id}/${action}`, {
        method: "POST",
        credentials: "include",
      });

      if (!response.ok) {
        const errorText = await response.text();
        showErrorToast(errorText || `Failed to ${action} account`);
        return;
      }

      showSuccessToast(
        account.installation_admin
          ? `${account.name} is no longer an installation admin`
          : `${account.name} is now an installation admin`,
      );
      fetchAccounts(search, offset);
    } catch {
      showErrorToast(`Failed to ${action} account`);
    } finally {
      setToggling(null);
    }
  };

  const handleImpersonate = async (orgId: string, userId: string) => {
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

  const totalPages = Math.ceil(total / PAGE_SIZE);
  const isPromoting = confirmTarget != null && !confirmTarget.installation_admin;

  return (
    <div>
      {/* Confirmation modal */}
      <Dialog open={confirmTarget !== null} onClose={() => setConfirmTarget(null)} size="md">
        {confirmTarget && (
          <>
            <div className="flex items-center gap-3 mb-2">
              <div
                className={`p-2 rounded-full ${isPromoting ? "bg-amber-100 text-amber-600" : "bg-red-100 text-red-600"}`}
              >
                <AlertTriangle size={20} />
              </div>
              <DialogTitle className="text-gray-800">
                {isPromoting ? "Promote to Installation Admin" : "Remove Installation Admin"}
              </DialogTitle>
            </div>

            <DialogDescription className="text-sm text-gray-600 mt-2 space-y-2">
              {isPromoting ? (
                <>
                  <p>
                    You are about to grant <strong>{confirmTarget.name}</strong> ({confirmTarget.email}) installation
                    admin access.
                  </p>
                  <p>This will allow them to:</p>
                  <ul className="list-disc pl-5 space-y-1 text-gray-500">
                    <li>View all organizations and their data across this installation</li>
                    <li>Impersonate any user in any organization</li>
                    <li>Promote or demote other installation admins</li>
                  </ul>
                </>
              ) : (
                <>
                  <p>
                    You are about to remove installation admin access from <strong>{confirmTarget.name}</strong> (
                    {confirmTarget.email}).
                  </p>
                  <p>They will lose the ability to access the admin dashboard and impersonate users.</p>
                </>
              )}
            </DialogDescription>

            <DialogActions>
              <Button variant={isPromoting ? "default" : "destructive"} onClick={() => executeToggle(confirmTarget)}>
                {isPromoting ? "Promote to Admin" : "Remove Admin Access"}
              </Button>
              <Button variant="outline" onClick={() => setConfirmTarget(null)}>
                Cancel
              </Button>
            </DialogActions>
          </>
        )}
      </Dialog>

      <div className="flex items-center justify-between mb-4">
        <div>
          <Heading className="text-gray-800 mb-0.5">Accounts</Heading>
          <Text className="text-gray-500 text-sm">
            {total} account{total !== 1 ? "s" : ""} across this installation
          </Text>
        </div>

        <div className="relative w-72">
          <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
          <input
            type="text"
            placeholder="Search by name or email..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="w-full pl-9 pr-3 py-1.5 text-sm border border-slate-200 rounded-md bg-white focus:outline-none focus:ring-1 focus:ring-blue-500 focus:border-blue-500"
          />
        </div>
      </div>

      {loading && accounts.length === 0 ? (
        <div className="flex flex-col items-center space-y-4 py-12">
          <div className="animate-spin rounded-full h-8 w-8 border-b border-gray-500"></div>
          <Text className="text-gray-500">Loading accounts...</Text>
        </div>
      ) : accounts.length === 0 ? (
        <div className="text-center py-12">
          <Text className="text-gray-500">
            {search ? "No accounts match your search." : "No accounts found."}
          </Text>
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
                {accounts.map((acc) => {
                  const isSelf = acc.id === currentAccount?.id;

                  return (
                    <tr key={acc.id} className="border-b border-slate-50 last:border-0">
                      <td className="px-4 py-2.5 text-gray-800">
                        {acc.name}
                        {isSelf && <span className="ml-1.5 text-xs text-gray-400">(you)</span>}
                      </td>
                      <td className="px-4 py-2.5 text-gray-500">{acc.email}</td>
                      <td className="px-4 py-2.5">
                        {acc.installation_admin ? (
                          <span className="inline-flex items-center gap-1 text-xs font-medium text-amber-700 bg-amber-50 px-2 py-0.5 rounded">
                            <Shield size={12} />
                            Admin
                          </span>
                        ) : (
                          <span className="text-xs text-gray-400">User</span>
                        )}
                      </td>
                      <td className="px-4 py-2.5 text-right">
                        <div className="flex items-center justify-end gap-2">
                          {/* Impersonate */}
                          {isSelf ? null : acc.memberships.length === 0 ? (
                            <Text className="text-xs text-gray-400">No orgs</Text>
                          ) : acc.memberships.length === 1 ? (
                            <Button
                              variant="outline"
                              size="sm"
                              onClick={() =>
                                handleImpersonate(acc.memberships[0].organization_id, acc.memberships[0].user_id)
                              }
                              disabled={impersonating === acc.memberships[0].user_id}
                            >
                              <span className="flex items-center gap-1">
                                <Eye size={14} />
                                {impersonating === acc.memberships[0].user_id ? "Starting..." : "Impersonate"}
                              </span>
                            </Button>
                          ) : (
                            <DropdownMenu>
                              <DropdownMenuTrigger asChild>
                                <Button variant="outline" size="sm" disabled={impersonating !== null}>
                                  <span className="flex items-center gap-1">
                                    <Eye size={14} />
                                    Impersonate
                                    <ChevronDown size={12} />
                                  </span>
                                </Button>
                              </DropdownMenuTrigger>
                              <DropdownMenuContent align="end">
                                {acc.memberships.map((m) => (
                                  <DropdownMenuItem
                                    key={m.organization_id}
                                    onClick={() => handleImpersonate(m.organization_id, m.user_id)}
                                  >
                                    {m.organization_name}
                                  </DropdownMenuItem>
                                ))}
                              </DropdownMenuContent>
                            </DropdownMenu>
                          )}

                          {/* Admin toggle */}
                          {isSelf ? (
                            <Text className="text-xs text-gray-400">Cannot change own access</Text>
                          ) : (
                            <Button
                              variant="outline"
                              size="sm"
                              onClick={() => setConfirmTarget(acc)}
                              disabled={toggling === acc.id}
                            >
                              {toggling === acc.id ? (
                                "Updating..."
                              ) : acc.installation_admin ? (
                                <span className="flex items-center gap-1">
                                  <ShieldOff size={14} />
                                  Demote
                                </span>
                              ) : (
                                <span className="flex items-center gap-1">
                                  <Shield size={14} />
                                  Promote
                                </span>
                              )}
                            </Button>
                          )}
                        </div>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>

          {totalPages > 1 && (
            <div className="flex items-center justify-between mt-4 text-sm text-gray-500">
              <Text>
                Showing {offset + 1}–{Math.min(offset + PAGE_SIZE, total)} of {total}
              </Text>
              <div className="flex gap-2">
                <button
                  onClick={() => {
                    const o = offset - PAGE_SIZE;
                    setOffset(o);
                    fetchAccounts(search, o);
                  }}
                  disabled={offset === 0}
                  className="px-3 py-1 rounded border border-slate-200 bg-white hover:bg-slate-50 disabled:opacity-40 disabled:cursor-not-allowed"
                >
                  Previous
                </button>
                <button
                  onClick={() => {
                    const o = offset + PAGE_SIZE;
                    setOffset(o);
                    fetchAccounts(search, o);
                  }}
                  disabled={Math.floor(offset / PAGE_SIZE) + 1 >= totalPages}
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

export default AccountsList;
