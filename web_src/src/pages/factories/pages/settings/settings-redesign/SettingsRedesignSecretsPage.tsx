import { useState } from "react";
import { Link } from "react-router";

import { PermissionTooltip } from "@/components/PermissionGate";
import { Button } from "@/components/ui/button";
import { usePermissions } from "@/contexts/usePermissions";
import { useSecrets } from "@/hooks/useSecrets";
import { usePageTitle } from "@/hooks/usePageTitle";
import { useOrganizationSettingsPaths } from "@/lib/organizationSettingsPaths";
import { CreateSecretDialog } from "@/ui/CreateSecretDialog";
import { KeyRound, Plus } from "lucide-react";

import { FactorySettingsPageFrame } from "../FactorySettingsCard";
import { useFactorySettingsLayout } from "../factorySettingsLayoutContext";
import { SettingsListRow } from "./settingsRedesignParts";

export function SettingsRedesignSecretsPage() {
  const { organizationId } = useFactorySettingsLayout();
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  const { data: secrets = [], isLoading } = useSecrets(organizationId, "DOMAIN_TYPE_ORGANIZATION");
  const settingsPaths = useOrganizationSettingsPaths(organizationId);
  const canCreate = canAct("secrets", "create");
  const [createOpen, setCreateOpen] = useState(false);

  usePageTitle(["Secrets"]);

  const sorted = [...secrets].sort((left, right) =>
    (left.metadata?.name || "").localeCompare(right.metadata?.name || ""),
  );

  return (
    <FactorySettingsPageFrame
      title="Secrets"
      subtitle="Store values that automations and canvases can read."
      actions={
        <PermissionTooltip
          allowed={canCreate || permissionsLoading}
          message="You do not have permission to create secrets."
        >
          <Button
            type="button"
            size="sm"
            disabled={!canCreate}
            onClick={() => setCreateOpen(true)}
            data-testid="secrets-create-btn"
          >
            <Plus className="size-3.5" aria-hidden />
            Create secret
          </Button>
        </PermissionTooltip>
      }
    >
      {isLoading ? <p className="text-[13px] text-muted-foreground">Loading secrets...</p> : null}
      {!isLoading && sorted.length === 0 ? (
        <p className="text-[13px] text-muted-foreground">No secrets yet.</p>
      ) : (
        <ul data-testid="settings-redesign-secrets">
          {sorted.map((secret) => {
            const secretId = secret.metadata?.id || "";
            const keyCount = Object.keys(secret.spec?.local?.data || {}).length;
            return (
              <SettingsListRow
                key={secretId}
                icon={<KeyRound className="size-4" aria-hidden />}
                title={
                  <Link to={settingsPaths.secretDetail(secretId)} data-testid="secrets-secret-link">
                    {secret.metadata?.name || "Unnamed secret"}
                  </Link>
                }
                subtitle={`${keyCount} key${keyCount === 1 ? "" : "s"}`}
              />
            );
          })}
        </ul>
      )}

      <CreateSecretDialog open={createOpen} onOpenChange={setCreateOpen} organizationId={organizationId} />
    </FactorySettingsPageFrame>
  );
}
