import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { installInstallationLicense, type InstallationLicense } from "@/lib/license";
import React, { useState } from "react";

type LicenseInstallFormProps = {
  submitLabel: string;
  onInstalled: (license: InstallationLicense) => void;
  secondaryAction?: React.ReactNode;
};

export const LicenseInstallForm: React.FC<LicenseInstallFormProps> = ({
  submitLabel,
  onInstalled,
  secondaryAction,
}) => {
  const [value, setValue] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [installing, setInstalling] = useState(false);

  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault();
    setError(null);
    setInstalling(true);

    try {
      const license = await installInstallationLicense(value);
      setValue("");
      onInstalled(license);
    } catch (installError) {
      setError(installError instanceof Error ? installError.message : "Failed to install the license");
    } finally {
      setInstalling(false);
    }
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-3">
      <div>
        <Label htmlFor="license-key" className="mb-2 block text-left">
          License key
        </Label>
        <Textarea
          id="license-key"
          data-testid="license-key-input"
          value={value}
          onChange={(event) => setValue(event.target.value)}
          placeholder="Paste the license key that you received from SuperPlane"
          spellCheck={false}
          autoComplete="off"
          aria-invalid={error ? true : undefined}
          className="min-h-28 font-mono text-xs break-all"
        />
        {error ? (
          <p role="alert" className="mt-2 text-xs text-red-600 dark:text-red-400">
            {error}
          </p>
        ) : null}
      </div>

      <div className="flex flex-wrap items-center gap-3">
        <Button type="submit" data-testid="license-install" disabled={installing || value.trim() === ""}>
          {installing ? "Verifying..." : submitLabel}
        </Button>
        {secondaryAction}
      </div>
    </form>
  );
};
