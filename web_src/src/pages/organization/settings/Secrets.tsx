import { Icon } from "@/components/Icon";
import { usePageTitle } from "@/hooks/usePageTitle";
import { useReportPageReady } from "@/hooks/useReportPageReady";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Link } from "@/components/Link/link";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/Table/table";
import { Button } from "@/components/ui/button";
import { usePermissions } from "@/contexts/usePermissions";
import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { useSecrets } from "@/hooks/useSecrets";
import { CreateSecretDialog } from "@/ui/CreateSecretDialog";
import type { SuperplaneSecretsSecret } from "@/api-client";

interface SecretsProps {
  organizationId: string;
}

interface CreateSecretButtonProps {
  canCreate: boolean;
  permissionsLoading: boolean;
  onClick: () => void;
  className?: string;
}

function CreateSecretButton({ canCreate, permissionsLoading, onClick, className }: CreateSecretButtonProps) {
  return (
    <PermissionTooltip allowed={canCreate || permissionsLoading} message="You don't have permission to create secrets.">
      <Button
        className={`flex items-center ${className ?? ""}`}
        onClick={onClick}
        disabled={!canCreate}
        data-testid="secrets-create-btn"
      >
        <Icon name="plus" />
        Create Secret
      </Button>
    </PermissionTooltip>
  );
}

function SecretsEmptyState(props: CreateSecretButtonProps) {
  return (
    <div className="flex min-h-96 flex-col items-center justify-center text-center">
      <div className="flex justify-center items-center text-gray-800">
        <Icon name="key" size="xl" />
      </div>
      <p className="mt-3 text-sm text-gray-800">Create your first secret</p>
      <CreateSecretButton {...props} className="mt-4" />
    </div>
  );
}

function SecretsTableRows({
  secrets,
  getDetailPath,
}: {
  secrets: SuperplaneSecretsSecret[];
  getDetailPath: (id: string) => string;
}) {
  return (
    <>
      {secrets.map((secret) => {
        const secretId = secret.metadata?.id || "";
        const keyCount = Object.keys(secret.spec?.local?.data || {}).length;
        return (
          <TableRow key={secretId} className="last:[&>td]:border-b-0">
            <TableCell>
              <div className="flex items-center gap-2">
                <Icon name="key" size="sm" className="text-gray-800" />
                <Link
                  href={getDetailPath(secretId)}
                  className="cursor-pointer text-sm !font-semibold text-gray-800 !underline underline-offset-2"
                  data-testid="secrets-secret-link"
                >
                  {secret.metadata?.name || "Unnamed Secret"}
                </Link>
              </div>
            </TableCell>
            <TableCell>
              <span className="text-sm text-gray-500 dark:text-gray-400">
                {keyCount} key{keyCount === 1 ? "" : "s"}
              </span>
            </TableCell>
          </TableRow>
        );
      })}
    </>
  );
}

export function Secrets({ organizationId }: SecretsProps) {
  usePageTitle(["Secrets"]);
  const navigate = useNavigate();
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);
  const canCreateSecrets = canAct("secrets", "create");

  const { data: secrets = [], isLoading } = useSecrets(organizationId, "DOMAIN_TYPE_ORGANIZATION");

  useReportPageReady(!isLoading && !permissionsLoading);

  const handleCreateClick = () => {
    if (!canCreateSecrets) return;
    setIsCreateModalOpen(true);
  };

  const getSecretDetailPath = (id: string) => `/${organizationId}/settings/secrets/${id}`;

  if (isLoading) {
    return (
      <div className="space-y-6 pt-6">
        <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-300 dark:border-gray-800 overflow-hidden">
          <div className="px-6 pb-6 min-h-96 flex justify-center items-center">
            <p className="text-gray-500 dark:text-gray-400">Loading secrets...</p>
          </div>
        </div>
      </div>
    );
  }

  const sortedSecrets = [...secrets].sort((a, b) => (a.metadata?.name || "").localeCompare(b.metadata?.name || ""));
  const createButtonProps = {
    canCreate: canCreateSecrets,
    permissionsLoading,
    onClick: handleCreateClick,
  };

  return (
    <div className="space-y-6 pt-6">
      <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-300 dark:border-gray-800 overflow-hidden">
        {sortedSecrets.length > 0 && (
          <div className="px-6 pt-6 pb-4 flex items-center justify-start">
            <CreateSecretButton {...createButtonProps} />
          </div>
        )}
        <div className="px-6 pb-6 min-h-96">
          {sortedSecrets.length === 0 ? (
            <SecretsEmptyState {...createButtonProps} />
          ) : (
            <Table dense>
              <TableHead>
                <TableRow>
                  <TableHeader>Secret name</TableHeader>
                  <TableHeader>Keys</TableHeader>
                </TableRow>
              </TableHead>
              <TableBody>
                <SecretsTableRows secrets={sortedSecrets} getDetailPath={getSecretDetailPath} />
              </TableBody>
            </Table>
          )}
        </div>
      </div>

      <CreateSecretDialog
        open={isCreateModalOpen}
        onOpenChange={setIsCreateModalOpen}
        organizationId={organizationId}
        onCreated={(created) => {
          if (created.id) {
            navigate(`/${organizationId}/settings/secrets/${created.id}`);
          }
        }}
      />
    </div>
  );
}
