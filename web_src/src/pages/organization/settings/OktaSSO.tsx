import { useEffect, useState } from "react";
import { useParams } from "react-router-dom";
import { usePageTitle } from "@/hooks/usePageTitle";
import { Field, Fieldset, Label } from "../../../components/Fieldset/fieldset";
import { Input } from "../../../components/Input/input";
import { Button } from "@/components/ui/button";
import { LoadingButton } from "@/components/ui/loading-button";
import { Badge } from "@/components/ui/badge";
import { Switch } from "@/ui/switch";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { usePermissions } from "@/contexts/PermissionsContext";
import { PermissionTooltip } from "@/components/PermissionGate";
import { usePublicBaseURL } from "@/hooks/useAuthConfig";
import {
  organizationsGetOktaIdpSettings,
  organizationsUpdateOktaIdpSettings,
  organizationsRotateOktaScimBearerToken,
} from "../../../api-client/sdk.gen";
import { withOrganizationHeader } from "../../../lib/withOrganizationHeader";
import type { OrganizationsOktaIdpSettings } from "../../../api-client/types.gen";

function copyToClipboard(text: string) {
  navigator.clipboard.writeText(text).catch(() => {});
}

function flashMessage(setter: (msg: string | null) => void, message: string, duration = 3000) {
  setter(message);
  setTimeout(() => setter(null), duration);
}

function getOktaURLs(baseURL: string, orgId: string) {
  return {
    acsURL: `${baseURL}/auth/okta/${orgId}/saml/acs`,
    entityID: `${baseURL}/auth/okta/${orgId}`,
    loginURL: `${baseURL}/auth/okta/${orgId}/saml/login`,
    scimBaseURL: `${baseURL}/api/v1/scim/${orgId}/v2`,
  };
}

function StatusMessage({ message }: { message: string | null }) {
  if (!message) return null;
  return (
    <span className={`text-sm ${message.includes("Failed") ? "text-red-600" : "text-green-600"}`}>
      {message}
    </span>
  );
}

function CertificateField({
  configured,
  disabled,
  value,
  onChange,
}: {
  configured: boolean;
  disabled: boolean;
  value: string;
  onChange: (v: string) => void;
}) {
  return (
    <Field className="space-y-1.5">
      <div className="flex items-center gap-2">
        <Label className="text-sm font-medium text-gray-700 dark:text-gray-300">X.509 Certificate</Label>
        {configured && (
          <Badge variant="secondary" className="text-xs">
            Configured
          </Badge>
        )}
      </div>
      <p className="text-xs text-gray-500 dark:text-gray-400">
        Paste the PEM certificate from Okta&apos;s SAML setup instructions.{" "}
        {configured && "Leave blank to keep the existing certificate."}
      </p>
      <textarea
        placeholder={configured ? "Leave blank to keep existing" : "MIIDtDCCApygAwIBAgI..."}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        disabled={disabled}
        rows={5}
        className="w-full max-w-sm rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 px-3 py-2 text-sm font-mono text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50"
      />
    </Field>
  );
}

export function OktaSSO() {
  const { organizationId } = useParams<{ organizationId: string }>();
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  const publicBaseURL = usePublicBaseURL();
  usePageTitle(["SSO"]);

  const [settings, setSettings] = useState<OrganizationsOktaIdpSettings | null>(null);
  const [loadError, setLoadError] = useState<string | null>(null);

  // SAML form state
  const [samlIdpSSOURL, setSamlIdpSSOURL] = useState("");
  const [samlIdpIssuer, setSamlIdpIssuer] = useState("");
  const [samlIdpCertificatePEM, setSamlIdpCertificatePEM] = useState("");
  const [samlEnabled, setSamlEnabled] = useState(true);
  const [samlSaving, setSamlSaving] = useState(false);
  const [samlMessage, setSamlMessage] = useState<string | null>(null);

  // SCIM form state
  const [scimEnabled, setScimEnabled] = useState(true);
  const [scimSaving, setScimSaving] = useState(false);
  const [scimMessage, setScimMessage] = useState<string | null>(null);
  const [rotatingToken, setRotatingToken] = useState(false);
  const [newToken, setNewToken] = useState<string | null>(null);

  const { acsURL, entityID, loginURL, scimBaseURL } = getOktaURLs(publicBaseURL, organizationId ?? "");

  useEffect(() => {
    if (!organizationId) return;
    organizationsGetOktaIdpSettings(withOrganizationHeader({ path: { id: organizationId } }))
      .then((res) => {
        const s = res.data?.settings;
        if (s) {
          setSettings(s);
          setSamlIdpSSOURL(s.samlIdpSsoUrl || "");
          setSamlIdpIssuer(s.samlIdpIssuer || "");
          setSamlEnabled(s.configured ? (s.samlEnabled ?? true) : true);
          setScimEnabled(s.configured ? (s.scimEnabled ?? true) : true);
        }
      })
      .catch(() => setLoadError("Failed to load SSO settings."));
  }, [organizationId]);

  const handleSamlSave = async () => {
    if (!canAct("org", "update") || !organizationId) return;
    setSamlSaving(true);
    try {
      const body: {
        samlIdpSsoUrl?: string;
        samlIdpIssuer?: string;
        samlIdpCertificatePem?: string;
        samlEnabled?: boolean;
      } = {
        samlIdpSsoUrl: samlIdpSSOURL.trim(),
        samlIdpIssuer: samlIdpIssuer.trim(),
        samlEnabled,
      };
      if (samlIdpCertificatePEM.trim()) {
        body.samlIdpCertificatePem = samlIdpCertificatePEM.trim();
      }
      const res = await organizationsUpdateOktaIdpSettings(
        withOrganizationHeader({ path: { id: organizationId }, body }),
      );
      if (res.data?.settings) {
        setSettings(res.data.settings);
        setSamlIdpCertificatePEM("");
      }
      flashMessage(setSamlMessage, "SAML settings saved.");
    } catch {
      flashMessage(setSamlMessage, "Failed to save SAML settings.");
    } finally {
      setSamlSaving(false);
    }
  };

  const handleScimSave = async () => {
    if (!canAct("org", "update") || !organizationId) return;
    setScimSaving(true);
    try {
      const res = await organizationsUpdateOktaIdpSettings(
        withOrganizationHeader({ path: { id: organizationId }, body: { scimEnabled } }),
      );
      if (res.data?.settings) {
        setSettings(res.data.settings);
      }
      flashMessage(setScimMessage, "SCIM settings saved.");
    } catch {
      flashMessage(setScimMessage, "Failed to save SCIM settings.");
    } finally {
      setScimSaving(false);
    }
  };

  const handleRotateToken = async () => {
    if (!canAct("org", "update") || !organizationId) return;
    setRotatingToken(true);
    try {
      const res = await organizationsRotateOktaScimBearerToken(
        withOrganizationHeader({ path: { id: organizationId } }),
      );
      if (res.data?.settings) {
        setSettings(res.data.settings);
      }
      if (res.data?.scimBearerToken) {
        setNewToken(res.data.scimBearerToken);
      }
    } catch {
      flashMessage(setScimMessage, "Failed to rotate SCIM token.");
    } finally {
      setRotatingToken(false);
    }
  };

  const { scimBearerTokenConfigured: tokenConfigured = false, samlIdpCertificateConfigured: certConfigured = false } =
    settings ?? {};

  return loadError ? (
    <p className="text-sm text-red-500 pt-6">{loadError}</p>
  ) : (
    <div className="space-y-6 pt-6 text-left">
      {/* SAML Section */}
      <Fieldset className="bg-white dark:bg-gray-800 rounded-lg border border-gray-300 dark:border-gray-800 p-6 space-y-5">
        <div>
          <p className="text-sm font-semibold text-gray-800 dark:text-white">SAML Sign-In</p>
          <p className="text-sm text-gray-500 dark:text-gray-400 mt-0.5">
            Allow members to log in via Okta SAML 2.0. Paste the values from Okta&apos;s{" "}
            <em>View SAML setup instructions</em> panel, then copy the ACS URL and SP Entity ID into your Okta
            app&apos;s SAML settings.
          </p>
        </div>

        <Field className="space-y-1.5">
          <Label className="text-sm font-medium text-gray-700 dark:text-gray-300">
            Identity Provider Single Sign-On URL
          </Label>
          <p className="text-xs text-gray-500 dark:text-gray-400">
            Paste the SSO URL from Okta&apos;s SAML setup instructions.
          </p>
          <Input
            type="text"
            placeholder="https://dev-12345.okta.com/app/superplane/exk.../sso/saml"
            value={samlIdpSSOURL}
            onChange={(e) => setSamlIdpSSOURL(e.target.value)}
            disabled={!canAct("org", "update")}
            className="max-w-sm"
          />
        </Field>

        <Field className="space-y-1.5">
          <Label className="text-sm font-medium text-gray-700 dark:text-gray-300">Identity Provider Issuer</Label>
          <p className="text-xs text-gray-500 dark:text-gray-400">
            Paste the issuer (Entity ID) from Okta&apos;s SAML setup instructions.
          </p>
          <Input
            type="text"
            placeholder="http://www.okta.com/exk..."
            value={samlIdpIssuer}
            onChange={(e) => setSamlIdpIssuer(e.target.value)}
            disabled={!canAct("org", "update")}
            className="max-w-sm"
          />
        </Field>

        <CertificateField
          configured={certConfigured}
          disabled={!canAct("org", "update")}
          value={samlIdpCertificatePEM}
          onChange={setSamlIdpCertificatePEM}
        />

        <Field className="space-y-1.5">
          <Label className="text-sm font-medium text-gray-700 dark:text-gray-300">ACS URL (Single Sign-On URL)</Label>
          <p className="text-xs text-gray-500 dark:text-gray-400">
            Paste this into your Okta app&apos;s <strong>Single sign on URL</strong> field.
          </p>
          <div className="flex items-center gap-2 max-w-sm">
            <Input type="text" value={acsURL} readOnly className="bg-gray-50 dark:bg-gray-700 text-gray-600" />
            <Button type="button" variant="outline" onClick={() => copyToClipboard(acsURL)}>
              Copy
            </Button>
          </div>
        </Field>

        <Field className="space-y-1.5">
          <Label className="text-sm font-medium text-gray-700 dark:text-gray-300">SP Entity ID (Audience URI)</Label>
          <p className="text-xs text-gray-500 dark:text-gray-400">
            Paste this into your Okta app&apos;s <strong>Audience URI (SP Entity ID)</strong> field.
          </p>
          <div className="flex items-center gap-2 max-w-sm">
            <Input type="text" value={entityID} readOnly className="bg-gray-50 dark:bg-gray-700 text-gray-600" />
            <Button type="button" variant="outline" onClick={() => copyToClipboard(entityID)}>
              Copy
            </Button>
          </div>
        </Field>

        <Field className="space-y-1.5">
          <Label className="text-sm font-medium text-gray-700 dark:text-gray-300">Employee login URL</Label>
          <p className="text-xs text-gray-500 dark:text-gray-400">
            Share this URL with employees to let them sign in via Okta.
          </p>
          <div className="flex items-center gap-2 max-w-sm">
            <Input type="text" value={loginURL} readOnly className="bg-gray-50 dark:bg-gray-700 text-gray-600" />
            <Button type="button" variant="outline" onClick={() => copyToClipboard(loginURL)}>
              Copy
            </Button>
          </div>
        </Field>

        <Field>
          <div className="flex items-center gap-3">
            <Switch
              checked={samlEnabled}
              onCheckedChange={setSamlEnabled}
              disabled={!canAct("org", "update")}
              aria-label="Enable SAML sign-in"
            />
            <Label className="text-sm font-medium text-gray-700 dark:text-gray-300">Enable SAML sign-in</Label>
          </div>
        </Field>

        <div className="flex items-center gap-4 pt-1">
          <PermissionTooltip
            allowed={canAct("org", "update") || permissionsLoading}
            message="You don't have permission to update SSO settings."
          >
            <LoadingButton
              type="button"
              onClick={handleSamlSave}
              disabled={!canAct("org", "update")}
              loading={samlSaving}
              loadingText="Saving..."
            >
              Save
            </LoadingButton>
          </PermissionTooltip>
          <StatusMessage message={samlMessage} />
        </div>
      </Fieldset>

      {/* SCIM Section */}
      <Fieldset className="bg-white dark:bg-gray-800 rounded-lg border border-gray-300 dark:border-gray-800 p-6 space-y-5">
        <div>
          <p className="text-sm font-semibold text-gray-800 dark:text-white">SCIM Provisioning</p>
          <p className="text-sm text-gray-500 dark:text-gray-400 mt-0.5">
            Automatically provision and deprovision members via Okta. Configure SCIM in your Okta app&apos;s{" "}
            <em>Provisioning</em> tab.
          </p>
        </div>

        <Field className="space-y-1.5">
          <Label className="text-sm font-medium text-gray-700 dark:text-gray-300">SCIM base URL</Label>
          <p className="text-xs text-gray-500 dark:text-gray-400">
            Paste this into your Okta SCIM connector base URL field.
          </p>
          <div className="flex items-center gap-2 max-w-sm">
            <Input type="text" value={scimBaseURL} readOnly className="bg-gray-50 dark:bg-gray-700 text-gray-600" />
            <Button type="button" variant="outline" onClick={() => copyToClipboard(scimBaseURL)}>
              Copy
            </Button>
          </div>
        </Field>

        <Field className="space-y-1.5">
          <div className="flex items-center gap-2">
            <Label className="text-sm font-medium text-gray-700 dark:text-gray-300">SCIM bearer token</Label>
            {tokenConfigured ? (
              <Badge variant="secondary" className="text-xs">
                Configured
              </Badge>
            ) : (
              <Badge variant="outline" className="text-xs">
                Not configured
              </Badge>
            )}
          </div>
          <PermissionTooltip
            allowed={canAct("org", "update") || permissionsLoading}
            message="You don't have permission to update SSO settings."
          >
            <LoadingButton
              type="button"
              variant="outline"
              onClick={handleRotateToken}
              disabled={!canAct("org", "update")}
              loading={rotatingToken}
              loadingText="Generating..."
            >
              {tokenConfigured ? "Rotate token" : "Generate token"}
            </LoadingButton>
          </PermissionTooltip>
        </Field>

        <Field>
          <div className="flex items-center gap-3">
            <Switch
              checked={scimEnabled}
              onCheckedChange={setScimEnabled}
              disabled={!canAct("org", "update")}
              aria-label="Enable SCIM provisioning"
            />
            <Label className="text-sm font-medium text-gray-700 dark:text-gray-300">Enable SCIM provisioning</Label>
          </div>
        </Field>

        <div className="flex items-center gap-4 pt-1">
          <PermissionTooltip
            allowed={canAct("org", "update") || permissionsLoading}
            message="You don't have permission to update SSO settings."
          >
            <LoadingButton
              type="button"
              onClick={handleScimSave}
              disabled={!canAct("org", "update")}
              loading={scimSaving}
              loadingText="Saving..."
            >
              Save
            </LoadingButton>
          </PermissionTooltip>
          <StatusMessage message={scimMessage} />
        </div>
      </Fieldset>

      {/* New token modal */}
      <Dialog
        open={!!newToken}
        onOpenChange={(open) => {
          if (!open) setNewToken(null);
        }}
      >
        <DialogContent showCloseButton={false}>
          <DialogHeader>
            <DialogTitle>SCIM bearer token generated</DialogTitle>
            <DialogDescription>Copy this token now — it will not be shown again.</DialogDescription>
          </DialogHeader>
          <div className="my-2">
            <Input
              type="text"
              value={newToken || ""}
              readOnly
              className="font-mono text-xs bg-gray-50 dark:bg-gray-700"
            />
          </div>
          <DialogFooter>
            <Button
              type="button"
              onClick={() => {
                if (newToken) copyToClipboard(newToken);
                setNewToken(null);
              }}
            >
              Copy and close
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
