import { Dialog, DialogActions, DialogDescription, DialogTitle } from "@/components/Dialog/dialog";
import { Button } from "@/components/ui/button";
import { enterpriseFeatureLabel, type LicenseGrantChange } from "@/lib/license";

const joinLabels = (labels: string[]) => {
  if (labels.length <= 1) return labels[0] ?? "";
  if (labels.length === 2) return `${labels[0]} and ${labels[1]}`;
  return `${labels.slice(0, -1).join(", ")}, and ${labels[labels.length - 1]}`;
};

const licenseRemovalMessage = (organizationCount: number) => {
  const noun = organizationCount === 1 ? "organization" : "organizations";
  return `Removing the license removes all Enterprise features from ${organizationCount} ${noun}. You must enable them again after you install a new license.`;
};

const licenseReplacementMessage = (grants: LicenseGrantChange[]) => {
  const labels = [...new Set(grants.map((grant) => enterpriseFeatureLabel(grant.feature_key)))];
  const count = new Set(grants.map((grant) => grant.organization_id)).size;
  const noun = count === 1 ? "organization" : "organizations";
  return `This license does not include ${joinLabels(labels)}. ${count} ${noun} will lose access to these features.`;
};

export function RemoveLicenseDialog({
  open,
  removing,
  organizationCount,
  onClose,
  onConfirm,
}: {
  open: boolean;
  removing: boolean;
  organizationCount: number;
  onClose: () => void;
  onConfirm: () => void;
}) {
  return (
    <Dialog open={open} onClose={onClose} size="md" role="dialog" aria-modal="true" data-testid="license-remove-dialog">
      <DialogTitle className="text-gray-800 dark:text-gray-100">Remove the license?</DialogTitle>
      <DialogDescription className="mt-2 text-sm text-gray-600 dark:text-gray-400">
        {licenseRemovalMessage(organizationCount)}
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
}

export function InstallLicenseDialog({
  grants,
  installing,
  onClose,
  onConfirm,
}: {
  grants: LicenseGrantChange[] | null;
  installing: boolean;
  onClose: () => void;
  onConfirm: () => void;
}) {
  return (
    <Dialog
      open={grants !== null}
      onClose={onClose}
      size="md"
      role="dialog"
      aria-modal="true"
      data-testid="license-install-dialog"
    >
      <DialogTitle className="text-gray-800 dark:text-gray-100">Install this license?</DialogTitle>
      <DialogDescription className="mt-2 text-sm text-gray-600 dark:text-gray-400">
        {grants ? licenseReplacementMessage(grants) : ""}
      </DialogDescription>
      <DialogActions>
        <Button onClick={onConfirm} disabled={installing}>
          {installing ? "Installing..." : "Install license"}
        </Button>
        <Button variant="outline" onClick={onClose}>
          Cancel
        </Button>
      </DialogActions>
    </Dialog>
  );
}
