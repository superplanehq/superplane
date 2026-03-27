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
import { withOrganizationHeader } from "../../../utils/withOrganizationHeader";
import type { OrganizationsOktaIdpSettings } from "../../../api-client/types.gen";

export function OktaSSO() {
  const { organizationId } = useParams<{ organizationId: string }>();
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  const publicBaseURL = usePublicBaseURL();
  usePageTitle(["SSO"]);

  const canUpdate = canAct("org", "update");

  const [settings, setSettings] = useState<OrganizationsOktaIdpSettings | null>(null);
  const [loadError, setLoadError] = useState<string | null>(null);

  // OIDC form state
  const [issuerBaseUrl, setIssuerBaseUrl] = useState("");
  const [oauthClientId, setOauthClientId] = useState("");
  const [oauthClientSecret, setOauthClientSecret] = useState("");
  const [oidcEnabled, setOidcEnabled] = useState(true);
  const [oidcSaving, setOidcSaving] = useState(false);
  const [oidcMessage, setOidcMessage] = useState<string | null>(null);

  // SCIM form state
  const [scimEnabled, setScimEnabled] = useState(true);
  const [scimSaving, setScimSaving] = useState(false);
  const [scimMessage, setScimMessage] = useState<string | null>(null);
  const [rotatingToken, setRotatingToken] = useState(false);
  const [newToken, setNewToken] = useState<string | null>(null);

  const redirectURI = `${publicBaseURL}/auth/okta/${organizationId}/callback`;
  const loginURL = `${publicBaseURL}/auth/okta/${organizationId}`;
  const scimBaseURL = `${publicBaseURL}/api/v1/scim/${organizationId}/v2`;

  useEffect(() => {
    if (!organizationId) return;
    organizationsGetOktaIdpSettings(withOrganizationHeader({ path: { id: organizationId } }))
      .then((res) => {
        const s = res.data?.settings;
        if (s) {
          setSettings(s);
          setIssuerBaseUrl(s.issuerBaseUrl || "");
          setOauthClientId(s.oauthClientId || "");
          setOidcEnabled(s.oidcEnabled ?? true);
          setScimEnabled(s.scimEnabled ?? true);
        }
      })
      .catch(() => setLoadError("Failed to load SSO settings."));
  }, [organizationId]);

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text).catch(() => {});
  };

  const flashMessage = (
    setter: (msg: string | null) => void,
    message: string,
    duration = 3000,
  ) => {
    setter(message);
    setTimeout(() => setter(null), duration);
  };

  const handleOidcSave = async () => {
    if (!canUpdate || !organizationId) return;
    setOidcSaving(true);
    try {
      const body: {
        issuerBaseUrl?: string;
        oauthClientId?: string;
        oauthClientSecret?: string;
        oidcEnabled?: boolean;
      } = {
        issuerBaseUrl: issuerBaseUrl.trim(),
        oauthClientId: oauthClientId.trim(),
        oidcEnabled,
      };
      if (oauthClientSecret.trim()) {
        body.oauthClientSecret = oauthClientSecret.trim();
      }
      const res = await organizationsUpdateOktaIdpSettings(
        withOrganizationHeader({ path: { id: organizationId }, body }),
      );
      if (res.data?.settings) {
        setSettings(res.data.settings);
        setOauthClientSecret("");
      }
      flashMessage(setOidcMessage, "OIDC settings saved.");
    } catch {
      flashMessage(setOidcMessage, "Failed to save OIDC settings.");
    } finally {
      setOidcSaving(false);
    }
  };

  const handleScimSave = async () => {
    if (!canUpdate || !organizationId) return;
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
    if (!canUpdate || !organizationId) return;
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

  if (loadError) {
    return <p className="text-sm text-red-500 pt-6">{loadError}</p>;
  }

  const tokenConfigured = settings?.scimBearerTokenConfigured ?? false;
  const secretConfigured = settings?.oauthClientSecretConfigured ?? false;

  return (
    <div className="space-y-6 pt-6 text-left">
      {/* OIDC Section */}
      <Fieldset className="bg-white dark:bg-gray-800 rounded-lg border border-gray-300 dark:border-gray-800 p-6 space-y-5">
        <div>
          <p className="text-sm font-semibold text-gray-800 dark:text-white">OIDC Sign-In</p>
          <p className="text-sm text-gray-500 dark:text-gray-400 mt-0.5">
            Allow members to log in via Okta.
          </p>
        </div>

        <Field className="space-y-1.5">
          <Label className="text-sm font-medium text-gray-700 dark:text-gray-300">Okta Issuer URL</Label>
          <Input
            type="text"
            placeholder="https://dev-12345.okta.com/oauth2/default"
            value={issuerBaseUrl}
            onChange={(e) => setIssuerBaseUrl(e.target.value)}
            disabled={!canUpdate}
            className="max-w-sm"
          />
        </Field>

        <Field className="space-y-1.5">
          <Label className="text-sm font-medium text-gray-700 dark:text-gray-300">Client ID</Label>
          <Input
            type="text"
            placeholder="Client ID"
            value={oauthClientId}
            onChange={(e) => setOauthClientId(e.target.value)}
            disabled={!canUpdate}
            className="max-w-sm"
          />
        </Field>

        <Field className="space-y-1.5">
          <div className="flex items-center gap-2">
            <Label className="text-sm font-medium text-gray-700 dark:text-gray-300">Client Secret</Label>
            {secretConfigured && (
              <Badge variant="secondary" className="text-xs">Configured</Badge>
            )}
          </div>
          <Input
            type="password"
            placeholder={secretConfigured ? "Leave blank to keep existing" : "Client Secret"}
            value={oauthClientSecret}
            onChange={(e) => setOauthClientSecret(e.target.value)}
            disabled={!canUpdate}
            className="max-w-sm"
          />
        </Field>

        <Field className="space-y-1.5">
          <Label className="text-sm font-medium text-gray-700 dark:text-gray-300">Sign-in redirect URI</Label>
          <p className="text-xs text-gray-500 dark:text-gray-400">Paste this into your Okta app's redirect URIs.</p>
          <div className="flex items-center gap-2 max-w-sm">
            <Input type="text" value={redirectURI} readOnly className="bg-gray-50 dark:bg-gray-700 text-gray-600" />
            <Button type="button" variant="outline" onClick={() => copyToClipboard(redirectURI)}>
              Copy
            </Button>
          </div>
        </Field>

        <Field className="space-y-1.5">
          <Label className="text-sm font-medium text-gray-700 dark:text-gray-300">Employee login URL</Label>
          <p className="text-xs text-gray-500 dark:text-gray-400">Share this URL with employees to let them sign in via Okta.</p>
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
              checked={oidcEnabled}
              onCheckedChange={setOidcEnabled}
              disabled={!canUpdate}
              aria-label="Enable OIDC sign-in"
            />
            <Label className="text-sm font-medium text-gray-700 dark:text-gray-300">
              Enable OIDC sign-in
            </Label>
          </div>
        </Field>

        <div className="flex items-center gap-4 pt-1">
          <PermissionTooltip
            allowed={canUpdate || permissionsLoading}
            message="You don't have permission to update SSO settings."
          >
            <LoadingButton
              type="button"
              onClick={handleOidcSave}
              disabled={!canUpdate}
              loading={oidcSaving}
              loadingText="Saving..."
            >
              Save
            </LoadingButton>
          </PermissionTooltip>
          {oidcMessage && (
            <span className={`text-sm ${oidcMessage.includes("Failed") ? "text-red-600" : "text-green-600"}`}>
              {oidcMessage}
            </span>
          )}
        </div>
      </Fieldset>

      {/* SCIM Section */}
      <Fieldset className="bg-white dark:bg-gray-800 rounded-lg border border-gray-300 dark:border-gray-800 p-6 space-y-5">
        <div>
          <p className="text-sm font-semibold text-gray-800 dark:text-white">SCIM Provisioning</p>
          <p className="text-sm text-gray-500 dark:text-gray-400 mt-0.5">
            Automatically provision and deprovision members via Okta.
          </p>
        </div>

        <Field className="space-y-1.5">
          <Label className="text-sm font-medium text-gray-700 dark:text-gray-300">SCIM base URL</Label>
          <p className="text-xs text-gray-500 dark:text-gray-400">Paste this into your Okta SCIM app configuration.</p>
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
              <Badge variant="secondary" className="text-xs">Configured</Badge>
            ) : (
              <Badge variant="outline" className="text-xs">Not configured</Badge>
            )}
          </div>
          <PermissionTooltip
            allowed={canUpdate || permissionsLoading}
            message="You don't have permission to update SSO settings."
          >
            <LoadingButton
              type="button"
              variant="outline"
              onClick={handleRotateToken}
              disabled={!canUpdate}
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
              disabled={!canUpdate}
              aria-label="Enable SCIM provisioning"
            />
            <Label className="text-sm font-medium text-gray-700 dark:text-gray-300">
              Enable SCIM provisioning
            </Label>
          </div>
        </Field>

        <div className="flex items-center gap-4 pt-1">
          <PermissionTooltip
            allowed={canUpdate || permissionsLoading}
            message="You don't have permission to update SSO settings."
          >
            <LoadingButton
              type="button"
              onClick={handleScimSave}
              disabled={!canUpdate}
              loading={scimSaving}
              loadingText="Saving..."
            >
              Save
            </LoadingButton>
          </PermissionTooltip>
          {scimMessage && (
            <span className={`text-sm ${scimMessage.includes("Failed") ? "text-red-600" : "text-green-600"}`}>
              {scimMessage}
            </span>
          )}
        </div>
      </Fieldset>

      {/* New token modal */}
      <Dialog open={!!newToken} onOpenChange={(open) => { if (!open) setNewToken(null); }}>
        <DialogContent showCloseButton={false}>
          <DialogHeader>
            <DialogTitle>SCIM bearer token generated</DialogTitle>
            <DialogDescription>
              Copy this token now — it will not be shown again.
            </DialogDescription>
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
