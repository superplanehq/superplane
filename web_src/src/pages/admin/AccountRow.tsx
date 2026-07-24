import { Text } from "@/components/Text/text";
import { Button } from "@/components/ui/button";
import { Ban, Shield, ShieldOff, ShieldCheck } from "lucide-react";
import React from "react";
import { formatDate } from "./formatDate";

interface AdminAccount {
  id: string;
  name: string;
  email: string;
  installation_admin: boolean;
  blocked: boolean;
  created_at?: string;
}

interface AccountRowProps {
  acc: AdminAccount;
  isSelf: boolean;
  toggling: boolean;
  onPromoteDemote: () => void;
  onBlockUnblock: () => void;
  impersonateButton: React.ReactNode;
}

export function AccountRow({
  acc,
  isSelf,
  toggling,
  onPromoteDemote,
  onBlockUnblock,
  impersonateButton,
}: AccountRowProps) {
  return (
    <tr className="border-b border-slate-50 last:border-0 dark:border-gray-800/70">
      <td className="px-4 py-2.5 text-gray-800 dark:text-gray-100">
        {acc.name || (
          <span className="text-gray-400 italic dark:text-gray-500" title={acc.id}>
            {acc.id.slice(0, 8)}...
          </span>
        )}
        {isSelf && <span className="ml-1.5 text-xs text-gray-400 dark:text-gray-500">(you)</span>}
      </td>
      <td className="px-4 py-2.5 text-gray-500 dark:text-gray-400">{acc.email}</td>
      <td className="px-4 py-2.5">
        <div className="flex flex-wrap items-center gap-1.5">
          {acc.installation_admin ? (
            <span className="inline-flex items-center gap-1 text-xs font-medium text-amber-700 bg-amber-50 px-2 py-0.5 rounded dark:bg-amber-950/40 dark:text-amber-300">
              <Shield size={12} />
              Admin
            </span>
          ) : (
            <span className="text-xs text-gray-400 dark:text-gray-500">User</span>
          )}
          {acc.blocked && (
            <span className="inline-flex items-center gap-1 text-xs font-medium text-red-700 bg-red-50 px-2 py-0.5 rounded dark:bg-red-950/40 dark:text-red-300">
              <Ban size={12} />
              Blocked
            </span>
          )}
        </div>
      </td>
      <td className="px-4 py-2.5 text-gray-400 text-xs whitespace-nowrap dark:text-gray-500">
        {formatDate(acc.created_at)}
      </td>
      <td className="px-4 py-2.5 text-right">
        <div className="flex items-center justify-end gap-2">
          {!isSelf && !acc.blocked && impersonateButton}
          {isSelf ? (
            <Text className="text-xs text-gray-400 dark:text-gray-500">Cannot change own access</Text>
          ) : (
            <>
              <Button variant="outline" size="sm" onClick={onBlockUnblock} disabled={toggling}>
                <BlockActionLabel toggling={toggling} blocked={acc.blocked} />
              </Button>
              <Button variant="outline" size="sm" onClick={onPromoteDemote} disabled={toggling || acc.blocked}>
                <AdminActionLabel toggling={toggling} installationAdmin={acc.installation_admin} />
              </Button>
            </>
          )}
        </div>
      </td>
    </tr>
  );
}

function BlockActionLabel({ toggling, blocked }: { toggling: boolean; blocked: boolean }) {
  if (toggling) {
    return "Updating...";
  }

  if (blocked) {
    return (
      <span className="flex items-center gap-1">
        <ShieldCheck size={14} />
        Unblock
      </span>
    );
  }

  return (
    <span className="flex items-center gap-1">
      <Ban size={14} />
      Block
    </span>
  );
}

function AdminActionLabel({ toggling, installationAdmin }: { toggling: boolean; installationAdmin: boolean }) {
  if (toggling) {
    return "Updating...";
  }

  if (installationAdmin) {
    return (
      <span className="flex items-center gap-1">
        <ShieldOff size={14} />
        Demote
      </span>
    );
  }

  return (
    <span className="flex items-center gap-1">
      <Shield size={14} />
      Promote
    </span>
  );
}
