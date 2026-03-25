import { showErrorToast, showSuccessToast } from "@/utils/toast";

interface AdminAccount {
  id: string;
  name: string;
  installation_admin: boolean;
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
