import { Button } from "@/components/ui/button";
import { LoadingButton } from "@/components/ui/loading-button";
import { Clock } from "lucide-react";

import { FIRST_RUN_COPY } from "./firstRunCopy";

const copy = FIRST_RUN_COPY.choose;

/**
 * The organization step of the GitHub card. Organizations come from the
 * owners of the repositories that SuperPlane can see. Choosing one only
 * filters the repository step.
 */
export function FirstRunOrganizationStep({
  organizations,
  pendingOrganizations,
  synchronizing,
  disabled,
  grantAccessDisabled,
  grantingAccess,
  onSelectOrganization,
  onGrantAccess,
}: {
  organizations: string[];
  pendingOrganizations: string[];
  synchronizing: boolean;
  disabled: boolean;
  grantAccessDisabled: boolean;
  grantingAccess: boolean;
  onSelectOrganization: (organization: string) => void;
  onGrantAccess: () => void;
}) {
  const hasPendingApprovals = pendingOrganizations.length > 0;
  return (
    <>
      {hasPendingApprovals ? <PendingApprovalRows organizations={pendingOrganizations} /> : null}
      {organizations.length === 0 ? (
        <OrganizationEmptyState
          synchronizing={synchronizing}
          hasPendingApprovals={hasPendingApprovals}
          disabled={grantAccessDisabled}
          grantingAccess={grantingAccess}
          onGrantAccess={onGrantAccess}
        />
      ) : (
        <OrganizationList
          organizations={organizations}
          disabled={disabled}
          grantAccessDisabled={grantAccessDisabled}
          grantingAccess={grantingAccess}
          onSelectOrganization={onSelectOrganization}
          onGrantAccess={onGrantAccess}
        />
      )}
    </>
  );
}

function OrganizationList({
  organizations,
  disabled,
  grantAccessDisabled,
  grantingAccess,
  onSelectOrganization,
  onGrantAccess,
}: {
  organizations: string[];
  disabled: boolean;
  grantAccessDisabled: boolean;
  grantingAccess: boolean;
  onSelectOrganization: (organization: string) => void;
  onGrantAccess: () => void;
}) {
  return (
    <div className="space-y-3 text-left" data-testid="first-run-github-organization-picker">
      {organizations.map((organization) => (
        <Button
          key={organization}
          type="button"
          className="w-full justify-start"
          disabled={disabled}
          onClick={() => onSelectOrganization(organization)}
          data-testid={`first-run-github-use-${organization}`}
        >
          {copy.useOrganization(organization)}
        </Button>
      ))}
      <div className="flex items-center justify-between gap-3 text-[13px] text-muted-foreground">
        <span>{copy.missingOrganization}</span>
        <LoadingButton
          type="button"
          variant="outline"
          size="sm"
          onClick={onGrantAccess}
          disabled={grantAccessDisabled}
          loading={grantingAccess}
          loadingText={copy.openingGitHub}
          data-testid="first-run-grant-access"
        >
          {copy.installAction}
        </LoadingButton>
      </div>
    </div>
  );
}

function OrganizationEmptyState({
  synchronizing,
  hasPendingApprovals,
  disabled,
  grantingAccess,
  onGrantAccess,
}: {
  synchronizing: boolean;
  hasPendingApprovals: boolean;
  disabled: boolean;
  grantingAccess: boolean;
  onGrantAccess: () => void;
}) {
  return (
    <div
      className="space-y-4 rounded-lg border border-dashed border-border px-4 py-5"
      data-testid="first-run-repositories-empty"
    >
      <div className="space-y-1 text-[13px]">
        <p className="font-medium text-foreground">{synchronizing ? copy.emptySynchronizingTitle : copy.emptyTitle}</p>
        <p className="text-muted-foreground">{copy.emptyBody}</p>
      </div>
      <LoadingButton
        type="button"
        className="w-full"
        variant={hasPendingApprovals ? "outline" : "default"}
        onClick={onGrantAccess}
        disabled={disabled}
        loading={grantingAccess}
        loadingText={copy.openingGitHub}
        data-testid="first-run-grant-access"
      >
        {hasPendingApprovals ? copy.installAnotherAction : copy.installAction}
      </LoadingButton>
    </div>
  );
}

function PendingApprovalRows({ organizations }: { organizations: string[] }) {
  return (
    <div className="space-y-2 text-left" data-testid="first-run-github-install-requested" role="status">
      {organizations.map((organization) => (
        <div
          key={organization}
          className="flex gap-3 rounded-md border border-border px-3 py-2.5 text-[13px]"
          data-testid="first-run-github-waiting-row"
        >
          <Clock className="mt-0.5 size-3.5 shrink-0 text-muted-foreground" aria-hidden />
          <div className="min-w-0 space-y-0.5">
            <p className="font-medium">{copy.installRequested(organization)}</p>
            <p className="text-pretty text-muted-foreground">
              {copy.installRequestedBody(organization)} {copy.installRequestedNext}
            </p>
          </div>
        </div>
      ))}
    </div>
  );
}
