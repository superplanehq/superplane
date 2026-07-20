import type { ApiKeysApiKey } from "@/api-client/types.gen";
import { Icon } from "@/components/Icon";
import { Link } from "@/components/Link/link";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/Table/table";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { KeyRound } from "lucide-react";
import {
  settingsEmptyStateIconClassName,
  settingsEmptyStateSubtitleClassName,
  settingsEmptyStateTitleClassName,
  settingsTableLinkClassName,
} from "./settingsPageStyles";

interface ApiKeysContentProps {
  sorted: ApiKeysApiKey[];
  canCreate: boolean;
  canDelete: boolean;
  permissionsLoading: boolean;
  deletePending: boolean;
  onCreateClick: () => void;
  onDelete: (id: string, name: string) => void;
  getDetailPath: (id: string) => string;
  scopeLabel: (canvasIds?: string[]) => string;
}

function formatDateTime(value?: string) {
  if (!value) return "Never";
  return new Date(value).toLocaleString();
}

function isExpired(value?: string) {
  return !!value && new Date(value).getTime() <= Date.now();
}

function tokenStatus(apiKey: ApiKeysApiKey) {
  if (!apiKey.hasToken) return "None";
  if (isExpired(apiKey.expiresAt)) return "Expired";
  return "Active";
}

export function ApiKeysContent({
  sorted,
  canCreate,
  canDelete,
  permissionsLoading,
  deletePending,
  onCreateClick,
  onDelete,
  getDetailPath,
  scopeLabel,
}: ApiKeysContentProps) {
  if (sorted.length === 0) {
    return (
      <div className="flex min-h-96 flex-col items-center justify-center text-center">
        <div className={cn("flex items-center justify-center", settingsEmptyStateIconClassName)}>
          <KeyRound size={32} />
        </div>
        <p className={settingsEmptyStateTitleClassName}>Create your first API key</p>
        <p className={settingsEmptyStateSubtitleClassName}>API keys provide programmatic access to SuperPlane.</p>
        <PermissionTooltip
          allowed={canCreate || permissionsLoading}
          message="You don't have permission to create API keys."
        >
          <Button
            className="mt-4 flex items-center"
            onClick={onCreateClick}
            disabled={!canCreate}
            data-testid="api-key-create-btn"
          >
            <Icon name="plus" />
            Create API Key
          </Button>
        </PermissionTooltip>
      </div>
    );
  }

  return (
    <Table dense>
      <TableHead>
        <TableRow>
          <TableHeader>Name</TableHeader>
          <TableHeader>Description</TableHeader>
          <TableHeader>Access</TableHeader>
          <TableHeader>Expires</TableHeader>
          <TableHeader>Created by</TableHeader>
          <TableHeader>Token</TableHeader>
          <TableHeader></TableHeader>
        </TableRow>
      </TableHead>
      <TableBody>
        {sorted.map((apiKey) => (
          <TableRow key={apiKey.id} className="last:[&>td]:border-b-0">
            <TableCell>
              <div className="flex items-center gap-2">
                <KeyRound size={16} className="text-gray-500 dark:text-gray-400" />
                <Link
                  href={getDetailPath(apiKey.id || "")}
                  className={settingsTableLinkClassName}
                  data-testid="api-key-link"
                >
                  {apiKey.name || "Unnamed"}
                </Link>
              </div>
            </TableCell>
            <TableCell>
              <span className="text-sm text-gray-500 dark:text-gray-400">{apiKey.description || "-"}</span>
            </TableCell>
            <TableCell>
              <span className="text-sm text-gray-500 dark:text-gray-400">{scopeLabel(apiKey.canvasIds)}</span>
            </TableCell>
            <TableCell>
              <span className="text-sm text-gray-500 dark:text-gray-400">{formatDateTime(apiKey.expiresAt)}</span>
            </TableCell>
            <TableCell>
              <span className="text-sm text-gray-500 dark:text-gray-400">
                {apiKey.createdByName ? apiKey.createdByName?.trim() : "-"}
              </span>
            </TableCell>
            <TableCell>
              <span className="text-sm text-gray-500 dark:text-gray-400">{tokenStatus(apiKey)}</span>
            </TableCell>
            <TableCell>
              <div className="flex justify-end">
                <PermissionTooltip
                  allowed={canDelete || permissionsLoading}
                  message="You don't have permission to delete API keys."
                >
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => onDelete(apiKey.id || "", apiKey.name || "")}
                    disabled={!canDelete || deletePending}
                    className="text-red-600 hover:text-red-700 dark:text-red-400 dark:hover:text-red-300"
                    data-testid="api-key-delete-btn"
                  >
                    <Icon name="trash-2" size="sm" />
                  </Button>
                </PermissionTooltip>
              </div>
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}
