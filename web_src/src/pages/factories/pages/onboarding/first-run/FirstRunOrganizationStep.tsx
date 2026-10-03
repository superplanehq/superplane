import { Button } from "@/components/ui/button";
import { LoadingButton } from "@/components/ui/loading-button";
import { Clock } from "lucide-react";

import { FIRST_RUN_COPY } from "./firstRunCopy";
import { FirstRunMissingAccessLine } from "./FirstRunMissingAccessLine";
import { FirstRunSkeletonRows } from "./FirstRunSkeletonRows";

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
  const missingAccess = (
    <FirstRunMissingAccessLine
      hint={copy.organizationWriteAccessHint}
      question={copy.missingOrganization}
      action={copy.installAction}
      disabled={grantAccessDisabled}
      loading={grantingAccess}
      onClick={onGrantAccess}
    />
  );
  const empty = organizations.length === 0;

  return (
    <>
      {hasPendingApprovals ? <PendingApprovalRows organizations={pendingOrganizations} /> : null}
      {empty && !synchronizing ? (
        <OrganizationEmptyState
          hasPendingApprovals={hasPendingApprovals}
          disabled={grantAccessDisabled}
          grantingAccess={grantingAccess}
          onGrantAccess={onGrantAccess}
        />
      ) : (
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
          {synchronizing ? (
            <FirstRunSkeletonRows
              count={empty ? 2 : 1}
              label={copy.loadingOrganizations}
              testId="first-run-organizations-loading"
            />
          ) : null}
          {missingAccess}
        </div>
      )}
    </>
  );
}

function OrganizationEmptyState({
  hasPendingApprovals,
  disabled,
  grantingAccess,
  onGrantAccess,
}: {
  hasPendingApprovals: boolean;
  disabled: boolean;
  grantingAccess: boolean;
  onGrantAccess: () => void;
}) {
  return (
    <div className="space-y-4 py-5" data-testid="first-run-repositories-empty">
      <div className="space-y-1 text-[13px]">
        <p className="font-medium text-foreground">{copy.emptyTitle}</p>
        <p className="text-muted-foreground">{copy.emptyBody}</p>
      </div>
      <LoadingButton
        type="button"
        className="w-full"
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
