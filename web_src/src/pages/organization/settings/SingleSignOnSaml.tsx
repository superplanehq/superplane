import { useEffect, useMemo, useState } from "react";
import { useParams } from "react-router-dom";
import { useOktaSettings } from "../../../hooks/useOktaSettings";
import { Field, Fieldset, Label, Description, FieldGroup } from "../../../components/Fieldset/fieldset";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";

const wizardSteps = ["saml", "scim"] as const;

type WizardStep = (typeof wizardSteps)[number];

export function SingleSignOnSaml() {
  const { organizationId } = useParams<{ organizationId: string }>();
  const [samlIssuer, setSamlIssuer] = useState("");
  const [samlCertificate, setSamlCertificate] = useState("");
  const [enforceSSO, setEnforceSSO] = useState(false);
  const [showToken, setShowToken] = useState(false);
  const [lastToken, setLastToken] = useState<string | null>(null);
  const [wizardStep, setWizardStep] = useState<WizardStep>("saml");
  const [isWizardOpen, setIsWizardOpen] = useState(false);
  const [wizardError, setWizardError] = useState<string | null>(null);
  const [copiedField, setCopiedField] = useState<"sso" | "scim" | null>(null);

  const { oktaSettings, isLoading, error, updateSettings, isUpdating, rotateToken, isRotating, rotatedToken } =
    useOktaSettings(organizationId || "");

  const hasSaml = Boolean(oktaSettings?.samlIssuer && oktaSettings?.samlCertificate);
  const hasScim = Boolean(oktaSettings?.hasScimToken);
  const isConfigured = hasSaml && hasScim;
  const showZeroState = !hasSaml && !hasScim && !isWizardOpen;

  useEffect(() => {
    if (oktaSettings) {
      setSamlIssuer(oktaSettings.samlIssuer || "");
      setSamlCertificate(oktaSettings.samlCertificate || "");
      setEnforceSSO(oktaSettings.enforceSso ?? false);
    }
  }, [oktaSettings?.samlIssuer, oktaSettings?.samlCertificate, oktaSettings?.enforceSso]);

  useEffect(() => {
    if (rotatedToken) {
      setLastToken(rotatedToken);
      setShowToken(true);
    }
  }, [rotatedToken]);

  const baseUrl = useMemo(() => {
    if (typeof window === "undefined") return "";
    const { protocol, host } = window.location;
    return `${protocol}//${host}`;
  }, []);

  const ssoUrl = useMemo(() => {
    if (!organizationId || !baseUrl) return "";
    return `${baseUrl}/orgs/${organizationId}/okta/auth`;
  }, [baseUrl, organizationId]);

  const scimBaseUrl = useMemo(() => {
    if (!organizationId || !baseUrl) return "";
    return `${baseUrl}/orgs/${organizationId}/okta/scim`;
  }, [baseUrl, organizationId]);

  const handleOpenWizard = () => {
    setWizardError(null);
    setShowToken(false);
    setLastToken(null);
    setWizardStep(hasSaml && !hasScim ? "scim" : "saml");
    setIsWizardOpen(true);
  };

  const handleCloseWizard = () => {
    setWizardError(null);
    setIsWizardOpen(false);
    setWizardStep("saml");
  };

  const handleSaveSaml = async () => {
    setWizardError(null);
    try {
      await updateSettings({
        settings: {
          samlIssuer,
          samlCertificate,
          enforceSso: enforceSSO,
          hasScimToken: oktaSettings?.hasScimToken ?? false,
        },
      });
      setWizardStep("scim");
    } catch (err) {
      setWizardError(err instanceof Error ? err.message : "Failed to save SAML settings.");
    }
  };

  const handleRotateToken = async () => {
    setWizardError(null);
    try {
      await rotateToken({});
    } catch (err) {
      setWizardError(err instanceof Error ? err.message : "Failed to generate SCIM token.");
    }
  };

  const handleFinishWizard = () => {
    setIsWizardOpen(false);
    setWizardStep("saml");
  };

  const handleCopy = async (value: string, field: "sso" | "scim") => {
    if (!value) return;
    try {
      await navigator.clipboard.writeText(value);
      setCopiedField(field);
      window.setTimeout(() => {
        setCopiedField((current) => (current === field ? null : current));
      }, 1500);
    } catch (err) {
      console.error("Failed to copy value:", err);
    }
  };

  if (isLoading) {
    return (
      <div className="pt-6">
        <p className="text-gray-500 dark:text-gray-400">Loading Okta settings...</p>
      </div>
    );
  }

  if (error) {
    return (
      <div className="pt-6">
        <p className="text-red-600 dark:text-red-400">Failed to load Okta settings.</p>
      </div>
    );
  }

  return (
    <div className="space-y-6 pt-6">
      <div className="bg-white dark:bg-gray-950 rounded-lg border border-gray-300 dark:border-gray-800 overflow-hidden">
        <div className="px-6 pt-6 pb-4 flex items-center justify-between">
          <div />
          {!showZeroState && !isWizardOpen && (
            <Button onClick={handleOpenWizard}>{isConfigured ? "Edit settings" : "Configure"}</Button>
          )}
        </div>

        {showZeroState && (
          <div className="px-6 pb-8">
            <div className="border border-dashed border-gray-300 dark:border-gray-800 rounded-lg p-8 text-center">
              <h3 className="text-base font-semibold text-gray-900 dark:text-gray-100">Set up Okta for your org</h3>
              <p className="text-sm text-gray-600 dark:text-gray-400 mt-2 max-w-md mx-auto">
                Connect Okta to enable secure SAML sign-on and SCIM provisioning. We will walk you through the exact
                values to copy from Okta and where to paste them.
              </p>
              <Button className="mt-6" onClick={handleOpenWizard}>
                Configure Okta
              </Button>
            </div>
          </div>
        )}

        {!showZeroState && !isWizardOpen && (
          <div className="px-6 pb-8 space-y-5">
            {!isConfigured && (
              <div className="border border-amber-200 dark:border-amber-800 bg-amber-50 dark:bg-amber-900/20 rounded-lg p-4">
                <p className="text-sm text-amber-900 dark:text-amber-100 font-medium">Finish setup</p>
                <p className="text-sm text-amber-800 dark:text-amber-200 mt-1">
                  SAML is saved. Add a SCIM token to finish provisioning.
                </p>
                <Button variant="outline" size="sm" className="mt-3" onClick={handleOpenWizard}>
                  Continue setup
                </Button>
              </div>
            )}
            <div className="space-y-6">
              <div>
                <p className="text-sm font-semibold text-gray-900 dark:text-gray-100">Status</p>
                <p className="mt-2 text-sm text-gray-700 dark:text-gray-300">
                  SAML {hasSaml ? "configured" : "not configured"}. SCIM token {hasScim ? "configured" : "missing"}.
                  Sign-in {oktaSettings?.enforceSso ? "required" : "optional"}.
                </p>
                <p className="mt-2 text-xs text-gray-500 dark:text-gray-400">
                  {oktaSettings?.enforceSso ? "Everyone must use SSO." : "People can still sign in without SSO."}
                </p>
              </div>
              <div>
                <p className="text-sm font-semibold text-gray-900 dark:text-gray-100">URLs</p>
                <div className="mt-3 space-y-4 text-sm text-gray-700 dark:text-gray-300">
                  <div className="flex items-start justify-between gap-4">
                    <div className="flex-1 space-y-1">
                      <p className="text-xs uppercase tracking-wide text-gray-500 dark:text-gray-400">
                        Single Sign-On URL
                      </p>
                      <p className="text-sm text-gray-900 dark:text-gray-100 break-all">{ssoUrl || "-"}</p>
                    </div>
                    <Button
                      variant="ghost"
                      size="sm"
                      type="button"
                      onClick={() => handleCopy(ssoUrl, "sso")}
                      disabled={!ssoUrl}
                    >
                      {copiedField === "sso" ? "Copied" : "Copy"}
                    </Button>
                  </div>
                  <div className="flex items-start justify-between gap-4">
                    <div className="flex-1 space-y-1">
                      <p className="text-xs uppercase tracking-wide text-gray-500 dark:text-gray-400">SCIM Base URL</p>
                      <p className="text-sm text-gray-900 dark:text-gray-100 break-all">{scimBaseUrl || "-"}</p>
                    </div>
                    <Button
                      variant="ghost"
                      size="sm"
                      type="button"
                      onClick={() => handleCopy(scimBaseUrl, "scim")}
                      disabled={!scimBaseUrl}
                    >
                      {copiedField === "scim" ? "Copied" : "Copy"}
                    </Button>
                  </div>
                </div>
              </div>
            </div>
          </div>
        )}

        {isWizardOpen && (
          <div className="px-6 pb-8">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-xs uppercase tracking-wide text-gray-500 dark:text-gray-400">Setup Wizard</p>
                <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100 mt-1">
                  {wizardStep === "saml" ? "SAML single sign-on" : "Step 2: Configure SCIM"}
                </h3>
                <p className="text-sm text-gray-600 dark:text-gray-400 mt-1">
                  {wizardStep === "saml"
                    ? "Manage your organization’s membership while adding another level of security with SAML."
                    : "Enable SCIM provisioning and generate a token for Okta."}
                </p>
              </div>
            </div>

            {wizardError && (
              <div className="mt-4 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 text-red-700 dark:text-red-200 px-4 py-3 rounded">
                {wizardError}
              </div>
            )}

            {wizardStep === "saml" && (
              <div className="mt-6 space-y-6">
                <Fieldset>
                  <FieldGroup>
                    <Field>
                      <Label>Sign on URL</Label>
                      <Description>Members are forwarded here when signing in to your organization.</Description>
                      <Input readOnly value={ssoUrl} />
                    </Field>
                    <Field>
                      <Label>SAML Issuer</Label>
                      <Description>Typically a unique URL generated by your identity provider.</Description>
                      <Input
                        value={samlIssuer}
                        onChange={(event) => setSamlIssuer(event.target.value)}
                        placeholder="https://example.okta.com/app/..."
                      />
                    </Field>
                    <Field>
                      <Label>Public certificate</Label>
                      <Description>Paste the X.509 certificate from your identity provider.</Description>
                      <Textarea
                        value={samlCertificate}
                        onChange={(event) => setSamlCertificate(event.target.value)}
                        placeholder="-----BEGIN CERTIFICATE-----"
                        className="min-h-[160px]"
                      />
                    </Field>
                  </FieldGroup>
                </Fieldset>

                <div className="flex items-center justify-between">
                  <Button variant="outline" onClick={handleCloseWizard}>
                    Cancel
                  </Button>
                  <Button onClick={handleSaveSaml} disabled={isUpdating || !samlIssuer || !samlCertificate}>
                    {isUpdating ? "Saving..." : "Save and continue"}
                  </Button>
                </div>
              </div>
            )}

            {wizardStep === "scim" && (
              <div className="mt-6 space-y-6">
                <div className="border border-gray-200 dark:border-gray-800 rounded-lg p-4">
                  <p className="text-sm font-medium text-gray-900 dark:text-gray-100">SCIM base URL</p>
                  <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                    Paste this into Okta provisioning settings as the SCIM connector base URL.
                  </p>
                  <p className="text-sm text-gray-900 dark:text-gray-100 break-all mt-3">{scimBaseUrl || "-"}</p>
                </div>

                <div className="border border-gray-200 dark:border-gray-800 rounded-lg p-4 space-y-3">
                  <p className="text-sm font-medium text-gray-900 dark:text-gray-100">SCIM token</p>
                  <p className="text-xs text-gray-500 dark:text-gray-400">
                    Generate a token, then paste it in Okta as the API token for provisioning.
                  </p>
                  <Button variant="outline" onClick={handleRotateToken} disabled={isRotating}>
                    {isRotating ? "Generating..." : "Generate token"}
                  </Button>
                  {oktaSettings?.hasScimToken && !lastToken && (
                    <p className="text-xs text-gray-500 dark:text-gray-400">A SCIM token is already configured.</p>
                  )}
                  {showToken && lastToken && (
                    <div className="mt-3 rounded-lg border border-emerald-200 bg-emerald-50 px-4 py-3 text-emerald-900 dark:border-emerald-900/60 dark:bg-emerald-900/20 dark:text-emerald-100">
                      <p className="text-sm font-semibold">New SCIM token</p>
                      <p className="mt-2 rounded-md bg-white/80 px-3 py-2 text-sm font-mono text-emerald-900 shadow-sm dark:bg-gray-900/60 dark:text-emerald-100 break-all">
                        {lastToken}
                      </p>
                      <p className="mt-2 text-xs text-emerald-800 dark:text-emerald-200">
                        Copy and store this token securely. You will not be able to see it again.
                      </p>
                    </div>
                  )}
                </div>

                <div className="flex items-center justify-between">
                  <Button variant="outline" onClick={() => setWizardStep("saml")}>
                    Back to SAML
                  </Button>
                  <Button onClick={handleFinishWizard} disabled={!hasScim && !lastToken}>
                    Finish
                  </Button>
                </div>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
