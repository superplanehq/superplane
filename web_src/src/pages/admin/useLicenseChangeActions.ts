import {
  installInstallationLicense,
  previewInstallationLicense,
  previewLicenseRemoval,
  type InstallationLicense as InstallationLicenseStatus,
  type LicenseGrantChange,
} from "@/lib/license";
import { showErrorToast } from "@/lib/toast";
import { useCallback, useState } from "react";

const installResult = (status: InstallationLicenseStatus) => {
  if (status.auto_grant_organization_id && status.auto_grant_organization_name) {
    return { kind: "auto" as const, name: status.auto_grant_organization_name };
  }
  if ((status.organization_count ?? 0) > 1 && status.event === "activation") {
    return { kind: "enable" as const };
  }
  return null;
};

export type LicenseInstallNote = ReturnType<typeof installResult>;

export function useLicenseChangeActions(handleInstalled: (status: InstallationLicenseStatus) => void) {
  const [confirmingRemoval, setConfirmingRemoval] = useState(false);
  const [removalOrganizations, setRemovalOrganizations] = useState(0);
  const [pendingRemovalGrants, setPendingRemovalGrants] = useState<LicenseGrantChange[] | null>(null);
  const [installingConfirmed, setInstallingConfirmed] = useState(false);
  const [pendingLicense, setPendingLicense] = useState<string | null>(null);
  const [formKey, setFormKey] = useState(0);
  const [installNote, setInstallNote] = useState<LicenseInstallNote>(null);

  const finishInstall = useCallback(
    (next: InstallationLicenseStatus) => {
      setInstallNote(installResult(next));
      setFormKey((key) => key + 1);
      handleInstalled(next);
    },
    [handleInstalled],
  );

  const installWithPreview = useCallback(async (license: string) => {
    const preview = await previewInstallationLicense(license);
    if (preview.removed_grants.length > 0) {
      setPendingLicense(license);
      setPendingRemovalGrants(preview.removed_grants);
      return null;
    }
    return installInstallationLicense(license);
  }, []);

  const confirmInstall = useCallback(async () => {
    if (!pendingLicense) return;
    setInstallingConfirmed(true);
    try {
      finishInstall(await installInstallationLicense(pendingLicense));
      setPendingLicense(null);
      setPendingRemovalGrants(null);
    } catch (error) {
      showErrorToast(error instanceof Error ? error.message : "Failed to install the license");
    } finally {
      setInstallingConfirmed(false);
    }
  }, [finishInstall, pendingLicense]);

  const requestRemoval = useCallback(async () => {
    try {
      const grants = await previewLicenseRemoval();
      setRemovalOrganizations(new Set(grants.map((grant) => grant.organization_id)).size);
      setConfirmingRemoval(true);
    } catch (error) {
      showErrorToast(error instanceof Error ? error.message : "Failed to remove the license");
    }
  }, []);

  return {
    confirmingRemoval,
    setConfirmingRemoval,
    removalOrganizations,
    pendingRemovalGrants,
    setPendingRemovalGrants,
    installingConfirmed,
    setPendingLicense,
    formKey,
    installNote,
    finishInstall,
    installWithPreview,
    confirmInstall,
    requestRemoval,
  };
}
