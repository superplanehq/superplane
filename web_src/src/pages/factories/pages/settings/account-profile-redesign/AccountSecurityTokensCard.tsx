import { Icon } from "@/components/Icon";
import { Button } from "@/components/ui/button";
import { Plus } from "lucide-react";

import { FactorySettingsCard } from "../FactorySettingsCard";
import type { AccountRedesignToken } from "./accountProfileRedesignMocks";

export function AccountSecurityTokensCard({
  tokens,
  onCreate,
  onRevokeToken,
}: {
  tokens: AccountRedesignToken[];
  onCreate: () => void;
  onRevokeToken: (id: string) => void;
}) {
  return (
    <FactorySettingsCard
      title="Personal tokens"
      data-testid="account-redesign-tokens"
      action={
        <Button size="sm" onClick={onCreate} data-testid="account-redesign-create-token">
          <Plus className="size-3.5" aria-hidden />
          Create token
        </Button>
      }
    >
      <p className="text-[12px] text-muted-foreground">
        This token acts as you. Organization API keys act as the organization.
      </p>
      {tokens.length === 0 ? (
        <div
          className="mt-4 flex flex-col items-center rounded-lg border border-dashed border-border px-6 py-10 text-center"
          data-testid="account-redesign-token-empty"
        >
          <Icon name="key-round" size="lg" className="text-muted-foreground" />
          <p className="mt-2 text-sm font-medium text-foreground">No personal tokens</p>
          <p className="mt-1 max-w-sm text-xs text-muted-foreground">
            Create a token to call the SuperPlane API from a script or the CLI.
          </p>
        </div>
      ) : (
        <table className="mt-4 w-full text-left" data-testid="account-redesign-token-list">
          <thead>
            <tr>
              <th className="border-b border-border pb-2 text-xs font-medium text-muted-foreground">Name</th>
              <th className="border-b border-border pb-2 text-xs font-medium text-muted-foreground">Created</th>
              <th className="border-b border-border pb-2 text-xs font-medium text-muted-foreground">Last used</th>
              <th className="border-b border-border pb-2" />
            </tr>
          </thead>
          <tbody>
            {tokens.map((token) => (
              <tr key={token.id}>
                <td className="py-2.5 text-[13px] text-foreground">{token.name}</td>
                <td className="py-2.5 text-[13px] text-muted-foreground">{token.createdAt}</td>
                <td className="py-2.5 text-[13px] text-muted-foreground">{token.lastUsedAt ?? "Never"}</td>
                <td className="py-2.5 text-right">
                  <Button type="button" size="sm" variant="ghost" onClick={() => onRevokeToken(token.id)}>
                    Revoke
                  </Button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </FactorySettingsCard>
  );
}
