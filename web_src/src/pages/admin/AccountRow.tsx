import { Text } from "@/components/Text/text";
import { Button } from "@/components/ui/button";
import { Shield, ShieldOff } from "lucide-react";
import React from "react";

interface AdminAccount {
  id: string;
  name: string;
  email: string;
  installation_admin: boolean;
}

interface AccountRowProps {
  acc: AdminAccount;
  isSelf: boolean;
  toggling: boolean;
  onPromoteDemote: () => void;
  impersonateButton: React.ReactNode;
}

export function AccountRow({ acc, isSelf, toggling, onPromoteDemote, impersonateButton }: AccountRowProps) {
  return (
    <tr className="border-b border-slate-50 last:border-0">
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
          {!isSelf && impersonateButton}
          {isSelf ? (
            <Text className="text-xs text-gray-400">Cannot change own access</Text>
          ) : (
            <Button variant="outline" size="sm" onClick={onPromoteDemote} disabled={toggling}>
              {toggling ? (
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
}
