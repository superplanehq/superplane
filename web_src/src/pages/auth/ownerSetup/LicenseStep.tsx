import { LicenseInstallForm } from "@/components/License/LicenseInstallForm";
import { Button } from "@/components/ui/button";
import { ENTERPRISE_FEATURES, fetchInstallationLicenseWhenKeysReady, type InstallationLicense } from "@/lib/license";
import { cn } from "@/lib/utils";
import { FirstRunHeading, FirstRunPanel } from "@/pages/factories/pages/onboarding/first-run/FirstRunShell";
import { Check } from "lucide-react";
import React, { useEffect, useState } from "react";

type LicenseStepProps = {
  onContinue: () => void;
};

type EditionChoice = "community" | "enterprise";

const featureLabels = (license: InstallationLicense) =>
  ENTERPRISE_FEATURES.filter((feature) => license.license?.features.includes(feature.key)).map(
    (feature) => feature.label,
  );

const ActivatedSummary = ({ license, onContinue }: { license: InstallationLicense; onContinue: () => void }) => (
  <div data-testid="owner-setup-license-active">
    <FirstRunHeading headline="SuperPlane Enterprise is active" size="display">
      <p className="text-[15px] leading-6 text-muted-foreground">
        Enabled features: {featureLabels(license).join(", ") || "none"}.
      </p>
    </FirstRunHeading>
    <Button type="button" className="mt-8 min-w-40" onClick={onContinue}>
      Continue
    </Button>
  </div>
);

const EditionOption = ({
  selected,
  title,
  description,
  testId,
  onSelect,
  children,
}: {
  selected: boolean;
  title: string;
  description: string;
  testId: string;
  onSelect: () => void;
  children?: React.ReactNode;
}) => (
  <div
    className={cn(
      "rounded-lg border px-4 py-3 text-left transition-colors",
      selected ? "border-foreground bg-accent/40" : "border-border bg-background",
    )}
  >
    <button
      type="button"
      role="radio"
      aria-checked={selected}
      data-testid={testId}
      onClick={onSelect}
      className="flex w-full items-start text-left"
    >
      <span className="min-w-0">
        <span className="flex items-center gap-2 text-[13px] font-medium tracking-[-0.01em]">
          {title}
          {selected ? <Check className="size-3.5" strokeWidth={2.5} aria-hidden /> : null}
        </span>
        <span className="mt-0.5 block text-[12px] text-muted-foreground">{description}</span>
      </span>
    </button>
    {children}
  </div>
);

export const LicenseStep: React.FC<LicenseStepProps> = ({ onContinue }) => {
  const [license, setLicense] = useState<InstallationLicense | null>(null);
  const [edition, setEdition] = useState<EditionChoice>("community");
  const [installing, setInstalling] = useState(false);
  const continued = React.useRef(false);

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
      <p className="text-[15px] text-muted-foreground" data-testid="owner-setup-license-loading">
        Preparing license verification...
      </p>
    );
  }

  if (license.edition === "enterprise") {
    return <ActivatedSummary license={license} onContinue={onContinue} />;
  }

  const continueWithCommunity = () => {
    if (installing) return;
    continued.current = true;
    onContinue();
  };

  return (
    <div data-testid="owner-setup-license">
      <FirstRunHeading headline="Choose an edition" size="display">
        <p className="text-[15px] leading-6 text-muted-foreground">
          Community is ready without a license. Add Enterprise only if you have one. You can add a license later in the
          installation admin settings.
        </p>
      </FirstRunHeading>

      <div className="mt-8">
        <FirstRunPanel>
          <div role="radiogroup" aria-label="Edition" className="space-y-3">
            <EditionOption
              selected={edition === "community"}
              title="Community"
              description="Start now. No license required."
              testId="owner-setup-edition-community"
              onSelect={() => setEdition("community")}
            />
            <EditionOption
              selected={edition === "enterprise"}
              title="Enterprise"
              description="Use a license file, or paste the key. This enables the features that license includes."
              testId="owner-setup-edition-enterprise"
              onSelect={() => setEdition("enterprise")}
            >
              {edition === "enterprise" ? (
                <div className="mt-4 border-t border-border pt-4">
                  <LicenseInstallForm
                    submitLabel="Activate Enterprise"
                    onInstallingChange={setInstalling}
                    onInstalled={(installed) => {
                      if (continued.current) return;
                      setLicense(installed);
                    }}
                  />
                </div>
              ) : null}
            </EditionOption>
          </div>
        </FirstRunPanel>
      </div>

      <Button
        type="button"
        variant={edition === "community" ? "default" : "ghost"}
        className={cn("mt-8 min-w-40", edition === "enterprise" && "text-muted-foreground hover:text-foreground")}
        data-testid="owner-setup-license-skip"
        disabled={installing}
        onClick={continueWithCommunity}
      >
        Continue with Community
      </Button>
    </div>
  );
};
