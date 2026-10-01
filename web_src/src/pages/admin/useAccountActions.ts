import { showErrorToast, showSuccessToast } from "@/lib/toast";

interface AdminAccount {
  id: string;
  name: string;
  installation_admin: boolean;
  blocked?: boolean;
}

export async function toggleAdmin(acc: AdminAccount, onDone: () => void) {
  const action = acc.installation_admin ? "demote" : "promote";
  try {
    const res = await fetch(`/admin/api/accounts/${acc.id}/${action}`, { method: "POST", credentials: "include" });
    if (!res.ok) {
      showErrorToast((await res.text()) || `Failed to ${action}`);
      return;
    }
    showSuccessToast(acc.installation_admin ? `${acc.name} removed as admin` : `${acc.name} promoted to admin`);
    onDone();
  } catch {
    showErrorToast(`Failed to ${action}`);
  }
}

export async function toggleBlock(acc: AdminAccount, onDone: () => void) {
  const action = acc.blocked ? "unblock" : "block";
  try {
    const res = await fetch(`/admin/api/accounts/${acc.id}/${action}`, { method: "POST", credentials: "include" });
    if (!res.ok) {
      showErrorToast((await res.text()) || `Failed to ${action}`);
      return;
    }
    showSuccessToast(acc.blocked ? `${acc.name} unblocked` : `${acc.name} blocked`);
    onDone();
  } catch {
    showErrorToast(`Failed to ${action}`);
  }
}

/** Returns the IDs of the organizations deleted with the account, or null when the delete fails. */
export async function deleteAccount(accountId: string, accountName: string): Promise<string[] | null> {
  try {
    const res = await fetch(`/admin/api/accounts/${accountId}`, { method: "DELETE", credentials: "include" });
    if (!res.ok) {
      showErrorToast((await res.text()) || "Failed to delete account");
      return null;
    }
    const body: { deleted_organization_ids?: string[] } = await res.json();
    showSuccessToast(`${accountName} deleted`);
    return body.deleted_organization_ids ?? [];
  } catch {
    showErrorToast("Failed to delete account");
    return null;
  }
}

export async function startImpersonation(accountId: string) {
  try {
    const res = await fetch("/admin/api/impersonate/start", {
      method: "POST",
      credentials: "include",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ account_id: accountId }),
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
}
