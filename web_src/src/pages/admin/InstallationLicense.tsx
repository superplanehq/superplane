import { Dialog, DialogActions, DialogDescription, DialogTitle } from "@/components/Dialog/dialog";
import { LicenseInstallForm } from "@/components/License/LicenseInstallForm";
import { Text } from "@/components/Text/text";
import { Button } from "@/components/ui/button";
import { useAccount } from "@/contexts/useAccount";
import { useReportPageReady } from "@/hooks/useReportPageReady";
import {
  ENTERPRISE_FEATURES,
  LICENSE_EXPIRY_WARNING_DAYS,
  daysUntil,
  fetchInstallationLicense,
  licenseReasonMessage,
  removeInstallationLicense,
  type InstallationLicense as InstallationLicenseStatus,
  type LicenseState,
} from "@/lib/license";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { CheckCircle2, Lock } from "lucide-react";
import React, { useCallback, useEffect, useState } from "react";
import { formatDate } from "./formatDate";

const stateBadges: Record<LicenseState, { label: string; className: string }> = {
  none: {
    label: "No license",
    className: "bg-slate-100 text-gray-600 dark:bg-gray-800 dark:text-gray-300",
  },
  active: {
    label: "Active",
    className: "bg-emerald-50 text-emerald-700 dark:bg-emerald-950/40 dark:text-emerald-300",
  },
  expired: {
    label: "Expired",
    className: "bg-red-50 text-red-700 dark:bg-red-950/40 dark:text-red-300",
  },
  not_yet_valid: {
    label: "Not valid yet",
    className: "bg-amber-50 text-amber-700 dark:bg-amber-950/40 dark:text-amber-300",
  },
  invalid: {
    label: "Invalid",
    className: "bg-red-50 text-red-700 dark:bg-red-950/40 dark:text-red-300",
  },
};

const sectionClass = "border-t border-slate-200 py-6 first:border-t-0 dark:border-gray-700/70";
const eyebrowClass = "text-xs font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400";

const StateBadge = ({ state }: { state: LicenseState }) => (
  <span
    data-testid="license-state"
    className={`inline-flex items-center rounded-md px-2 py-0.5 text-xs font-medium ${stateBadges[state].className}`}
  >
    {stateBadges[state].label}
  </span>
);

const DetailRow = ({ label, children }: { label: string; children: React.ReactNode }) => (
  <div>
    <dt className="text-xs text-gray-500 dark:text-gray-400">{label}</dt>
    <dd className="mt-1 text-sm text-gray-900 dark:text-gray-100">{children}</dd>
  </div>
);

const ExpiryValue = ({ status }: { status: InstallationLicenseStatus }) => {
  const expiresAt = status.license?.expires_at;
  if (!expiresAt) {
    return <>—</>;
  }

  const daysLeft = daysUntil(expiresAt);
  const showWarning = status.state === "active" && daysLeft <= LICENSE_EXPIRY_WARNING_DAYS;

  return (
    <span className="flex flex-wrap items-center gap-2">
      {formatDate(expiresAt)}
      {showWarning ? (
        <span className="rounded-md bg-amber-50 px-2 py-0.5 text-xs font-medium text-amber-700 dark:bg-amber-950/40 dark:text-amber-300">
          {daysLeft <= 1 ? "Expires today" : `Expires in ${daysLeft} days`}
        </span>
      ) : null}
    </span>
  );
};

const CurrentLicenseSection = ({ status }: { status: InstallationLicenseStatus }) => (
  <section className={sectionClass}>
    <p className={eyebrowClass}>Current plan</p>
    <div className="mt-1 flex flex-wrap items-center gap-3">
      <h2 className="text-base font-semibold text-gray-900 dark:text-gray-100">
        {status.edition === "enterprise" ? "SuperPlane Enterprise" : "SuperPlane Community"}
      </h2>
      <StateBadge state={status.state} />
    </div>

    {status.state === "invalid" ? (
      <Text className="mt-2 text-sm text-red-700 dark:text-red-400">{licenseReasonMessage(status.reason)}</Text>
    ) : null}

    {status.state === "expired" ? (
      <Text className="mt-2 text-sm text-gray-600 dark:text-gray-400">
        Enterprise features are not available. Existing custom roles and groups continue to work, but you cannot create
        new ones. Install a renewed license to enable them again.
      </Text>
    ) : null}

    {status.license ? (
      <dl className="mt-5 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <DetailRow label="License ID">
          <span className="font-mono text-xs">{status.license.id}</span>
        </DetailRow>
        <DetailRow label="Customer ID">
          <span className="font-mono text-xs">{status.license.customer_id}</span>
        </DetailRow>
        <DetailRow label="Valid from">{formatDate(status.license.valid_from)}</DetailRow>
        <DetailRow label="Expires">
          <ExpiryValue status={status} />
        </DetailRow>
      </dl>
    ) : null}

    {status.state === "none" ? (
      <Text className="mt-2 text-sm text-gray-600 dark:text-gray-400">
        Install an Enterprise license to enable Enterprise features for every organization in this installation.
      </Text>
    ) : null}
  </section>
);

const FeaturesSection = ({ status }: { status: InstallationLicenseStatus }) => {
  const granted = new Set(status.edition === "enterprise" ? (status.license?.features ?? []) : []);

  return (
    <section className={sectionClass}>
      <p className={eyebrowClass}>Enterprise features</p>
      <ul className="mt-3 grid gap-3 sm:grid-cols-2" data-testid="license-features">
        {ENTERPRISE_FEATURES.map((feature) => {
          const enabled = granted.has(feature.key);
          return (
            <li key={feature.key} className="flex items-start gap-2">
              {enabled ? (
                <CheckCircle2 size={16} className="mt-0.5 shrink-0 text-emerald-600 dark:text-emerald-400" />
              ) : (
                <Lock size={16} className="mt-0.5 shrink-0 text-gray-400 dark:text-gray-500" />
              )}
              <div>
                <p className="text-sm font-medium text-gray-900 dark:text-gray-100">
                  {feature.label}
                  <span className="sr-only">{enabled ? " (included)" : " (not included)"}</span>
                </p>
                <Text className="text-xs text-gray-500 dark:text-gray-400">{feature.description}</Text>
              </div>
            </li>
          );
        })}
      </ul>
    </section>
  );
};

const ManagedByConfigurationNote = () => (
  <section className={sectionClass}>
    <p className={eyebrowClass}>Managed license</p>
    <Text className="mt-2 max-w-2xl text-sm text-gray-600 dark:text-gray-400">
      The installation configuration provides this license through <code>SUPERPLANE_LICENSE_PATH</code>. To change it,
      update the license file or Kubernetes secret. SuperPlane reloads it within a minute.
    </Text>
  </section>
);

type RemoveLicenseDialogProps = {
  open: boolean;
  removing: boolean;
  onClose: () => void;
  onConfirm: () => void;
};

const RemoveLicenseDialog = ({ open, removing, onClose, onConfirm }: RemoveLicenseDialogProps) => (
  <Dialog open={open} onClose={onClose} size="md">
    <DialogTitle className="text-gray-800 dark:text-gray-100">Remove the license?</DialogTitle>
    <DialogDescription className="mt-2 text-sm text-gray-600 dark:text-gray-400">
      This installation returns to SuperPlane Community. Existing custom roles and groups continue to work, but nobody
      can create or change them until you install a license again.
    </DialogDescription>
    <DialogActions>
      <Button variant="destructive" onClick={onConfirm} disabled={removing}>
        {removing ? "Removing..." : "Remove license"}
      </Button>
      <Button variant="outline" onClick={onClose}>
        Cancel
      </Button>
    </DialogActions>
  </Dialog>
);

type InstallSectionProps = {
  status: InstallationLicenseStatus;
  onInstalled: (status: InstallationLicenseStatus) => void;
  onRemoveClick: () => void;
};

const InstallSection = ({ status, onInstalled, onRemoveClick }: InstallSectionProps) => {
  const hasStoredLicense = status.source === "database" && status.state !== "none";

  return (
    <section className={sectionClass}>
      <p className={eyebrowClass}>{hasStoredLicense ? "Replace license" : "Install license"}</p>
      <Text className="mt-2 mb-4 max-w-2xl text-sm text-gray-600 dark:text-gray-400">
        SuperPlane verifies the license on this server. It does not send the license to SuperPlane. To renew or add
        features, install the new license that you receive.
      </Text>
      <div className="max-w-2xl">
        <LicenseInstallForm
          submitLabel={hasStoredLicense ? "Replace license" : "Install license"}
          onInstalled={onInstalled}
          secondaryAction={
            hasStoredLicense ? (
              <Button type="button" variant="outline" data-testid="license-remove" onClick={onRemoveClick}>
                Remove license
              </Button>
            ) : null
          }
        />
      </div>
    </section>
  );
};

const useInstallationLicense = () => {
  const { refreshAccount } = useAccount();
  const [status, setStatus] = useState<InstallationLicenseStatus | null>(null);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [removing, setRemoving] = useState(false);

  useEffect(() => {
    fetchInstallationLicense()
      .then(setStatus)
      .catch((error: unknown) => setLoadError(error instanceof Error ? error.message : "Failed to load the license"));
  }, []);

  const handleInstalled = useCallback(
    (next: InstallationLicenseStatus) => {
      setStatus(next);
      showSuccessToast("License installed");
      void refreshAccount();
    },
    [refreshAccount],
  );

  const remove = useCallback(async () => {
    setRemoving(true);
    try {
      setStatus(await removeInstallationLicense());
      showSuccessToast("License removed");
      void refreshAccount();
    } catch (error) {
      showErrorToast(error instanceof Error ? error.message : "Failed to remove the license");
    } finally {
      setRemoving(false);
    }
  }, [refreshAccount]);

  return { status, loadError, removing, handleInstalled, remove };
};

const InstallationLicense: React.FC = () => {
  const { status, loadError, removing, handleInstalled, remove } = useInstallationLicense();
  const [confirmingRemoval, setConfirmingRemoval] = useState(false);

  useReportPageReady(status !== null || loadError !== null);

  return (
    <div>
      <div className="pb-2">
        <h1 className="text-xl font-semibold text-gray-900 dark:text-gray-100">License</h1>
        <Text className="mt-1 text-sm text-gray-500 dark:text-gray-400">
          Manage the Enterprise license for this installation. The license applies to every organization.
        </Text>
      </div>

      {loadError ? <Text className="mt-5 text-sm text-red-700 dark:text-red-400">{loadError}</Text> : null}

      {!status && !loadError ? (
        <div className="flex flex-col items-center space-y-4 py-12">
          <div className="h-8 w-8 animate-spin rounded-full border-b border-gray-500 dark:border-gray-400"></div>
          <Text className="text-gray-500 dark:text-gray-400">Loading license...</Text>
        </div>
      ) : null}

      {status ? (
        <div className="mt-5 bg-white px-6 dark:bg-gray-900">
          <CurrentLicenseSection status={status} />
          <FeaturesSection status={status} />
          {status.managed_by_configuration ? (
            <ManagedByConfigurationNote />
          ) : (
            <InstallSection
              status={status}
              onInstalled={handleInstalled}
              onRemoveClick={() => setConfirmingRemoval(true)}
            />
          )}
        </div>
      ) : null}

      <RemoveLicenseDialog
        open={confirmingRemoval}
        removing={removing}
        onClose={() => setConfirmingRemoval(false)}
        onConfirm={() => {
          void remove().then(() => setConfirmingRemoval(false));
        }}
      />
    </div>
  );
};

export default InstallationLicense;
