import superplaneLogo from "@/assets/superplane.svg";
import { LicenseInstallForm } from "@/components/License/LicenseInstallForm";
import { Text } from "@/components/Text/text";
import { Button } from "@/components/ui/button";
import { ENTERPRISE_FEATURES, fetchInstallationLicenseWhenKeysReady, type InstallationLicense } from "@/lib/license";
import { CheckCircle2 } from "lucide-react";
import React, { useEffect, useState } from "react";

type LicenseStepProps = {
  onContinue: () => void;
};

const featureLabels = (license: InstallationLicense) =>
  ENTERPRISE_FEATURES.filter((feature) => license.license?.features.includes(feature.key)).map(
    (feature) => feature.label,
  );

const ActivatedSummary = ({ license, onContinue }: { license: InstallationLicense; onContinue: () => void }) => (
  <div className="space-y-4 text-center" data-testid="owner-setup-license-active">
    <CheckCircle2 size={32} className="mx-auto text-emerald-600 dark:text-emerald-400" />
    <div>
      <h4 className="mb-1 text-xl font-medium text-gray-800 dark:text-white">SuperPlane Enterprise is active</h4>
      <Text className="text-gray-800 dark:text-gray-300">
        Enabled features: {featureLabels(license).join(", ") || "none"}.
      </Text>
    </div>
    <Button type="button" className="w-full" onClick={onContinue}>
      Continue
    </Button>
  </div>
);

export const LicenseStep: React.FC<LicenseStepProps> = ({ onContinue }) => {
  const [license, setLicense] = useState<InstallationLicense | null>(null);

  useEffect(() => {
    fetchInstallationLicenseWhenKeysReady()
      .then((status) => {
        if (status.managed_by_configuration) {
          onContinue();
          return;
        }
        setLicense(status);
      })
      .catch(onContinue);
  }, [onContinue]);

  if (!license) {
    return (
      <div className="flex flex-col items-center space-y-4 py-8" data-testid="owner-setup-license-loading">
        <div className="h-8 w-8 animate-spin rounded-full border-b border-gray-500 dark:border-gray-400"></div>
        <Text className="text-gray-500 dark:text-gray-400">Preparing license verification...</Text>
      </div>
    );
  }

  if (license.edition === "enterprise") {
    return <ActivatedSummary license={license} onContinue={onContinue} />;
  }

  return (
    <div data-testid="owner-setup-license">
      <div className="mb-6 text-center">
        <img
          src={superplaneLogo}
          alt="SuperPlane logo"
          className="mx-auto mb-4 h-8 w-8 dark:brightness-0 dark:invert"
        />
        <h4 className="mb-1 text-xl font-medium text-gray-800 dark:text-white">Activate SuperPlane Enterprise</h4>
        <Text className="text-gray-800 dark:text-gray-300">
          Optional. Paste your license key to enable Enterprise features. You can also do this later in the installation
          admin settings.
        </Text>
      </div>
      <LicenseInstallForm
        submitLabel="Activate"
        onInstalled={setLicense}
        secondaryAction={
          <Button type="button" variant="outline" data-testid="owner-setup-license-skip" onClick={onContinue}>
            Skip for now
          </Button>
        }
      />
    </div>
  );
};
