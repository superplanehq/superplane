import React, { useMemo, useState } from "react";
import { useParams } from "react-router-dom";
import { useOktaSettings } from "../../../hooks/useOktaSettings";
import { Text } from "../../../components/Text/text";
import { Button } from "../../../ui/button";
import { Input } from "../../../ui/input";
import { Checkbox } from "../../../ui/checkbox";

export function OktaIntegration() {
  const { organizationId } = useParams<{ organizationId: string }>();
  const [samlIssuer, setSamlIssuer] = useState("");
  const [samlCertificate, setSamlCertificate] = useState("");
  const [enforceSSO, setEnforceSSO] = useState(false);
  const [showToken, setShowToken] = useState(false);
  const [lastToken, setLastToken] = useState<string | null>(null);

  const {
    oktaSettings,
    isLoading,
    error,
    updateSettings,
    isUpdating,
    rotateToken,
    isRotating,
    rotatedToken,
  } = useOktaSettings(organizationId || "");

  React.useEffect(() => {
    if (oktaSettings) {
      setSamlIssuer(oktaSettings.samlIssuer || "");
      setSamlCertificate(oktaSettings.samlCertificate || "");
      setEnforceSSO(oktaSettings.enforceSso ?? false);
    }
  }, [oktaSettings?.samlIssuer, oktaSettings?.samlCertificate, oktaSettings?.enforceSso]);

  React.useEffect(() => {
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

  const handleSave = async () => {
    await updateSettings({
      settings: {
        samlIssuer,
        samlCertificate,
        enforceSso: enforceSSO,
        hasScimToken: oktaSettings?.hasScimToken ?? false,
      },
    });
  };

  const handleRotateToken = async () => {
    await rotateToken({});
  };

  if (isLoading) {
    return (
      <div className="pt-6">
        <Text className="text-gray-600 dark:text-gray-400">Loading Okta settings...</Text>
      </div>
    );
  }

  if (error) {
    return (
      <div className="pt-6">
        <Text className="text-red-600 dark:text-red-400">Failed to load Okta settings.</Text>
      </div>
    );
  }

  return (
    <div className="pt-6 max-w-4xl">
      <div className="mb-6">
        <h1 className="text-2xl font-semibold text-gray-900 dark:text-gray-100">Okta Integration</h1>
        <p className="text-gray-600 dark:text-gray-400 mt-2">
          Configure SAML Single Sign On and SCIM provisioning for this organization using Okta.
        </p>
      </div>

      <div className="bg-white dark:bg-gray-900 rounded-lg border border-gray-200 dark:border-gray-800 p-6 mb-6">
        <h2 className="text-lg font-medium text-gray-900 dark:text-gray-100 mb-4">SAML configuration</h2>

        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">SAML Issuer</label>
            <Input
              value={samlIssuer}
              onChange={(event) => setSamlIssuer(event.target.value)}
              placeholder="https://example.okta.com/app/..."
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              SAML Certificate
            </label>
            <textarea
              className="w-full min-h-[160px] rounded-md border border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-900 px-3 py-2 text-sm text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
              value={samlCertificate}
              onChange={(event) => setSamlCertificate(event.target.value)}
              placeholder="-----BEGIN CERTIFICATE-----"
            />
          </div>
          <div className="flex items-center gap-2">
            <Checkbox id="enforce-sso" checked={enforceSSO} onCheckedChange={(value) => setEnforceSSO(!!value)} />
            <label htmlFor="enforce-sso" className="text-sm text-gray-700 dark:text-gray-300">
              Enforce Okta SSO for this organization
            </label>
          </div>
          <div>
            <Button onClick={handleSave} disabled={isUpdating}>
              {isUpdating ? "Saving..." : "Save settings"}
            </Button>
          </div>
        </div>
      </div>

      <div className="bg-white dark:bg-gray-900 rounded-lg border border-gray-200 dark:border-gray-800 p-6 mb-6">
        <h2 className="text-lg font-medium text-gray-900 dark:text-gray-100 mb-4">URLs for Okta</h2>
        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              Single Sign On URL
            </label>
            <Input readOnly value={ssoUrl} />
            <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
              Use this as the Single Sign On URL in your Okta SAML application.
            </p>
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              SCIM base URL
            </label>
            <Input readOnly value={scimBaseUrl} />
            <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
              Use this as the SCIM connector base URL in your Okta provisioning settings.
            </p>
          </div>
        </div>
      </div>

      <div className="bg-white dark:bg-gray-900 rounded-lg border border-gray-200 dark:border-gray-800 p-6">
        <h2 className="text-lg font-medium text-gray-900 dark:text-gray-100 mb-4">SCIM token</h2>
        <p className="text-sm text-gray-600 dark:text-gray-400 mb-4">
          Generate a SCIM token to authenticate Okta when provisioning users and groups. The token will be shown only
          once after rotation.
        </p>
        <div className="flex items-center gap-3">
          <Button variant="outline" onClick={handleRotateToken} disabled={isRotating}>
            {isRotating ? "Generating..." : "Generate new token"}
          </Button>
          {oktaSettings?.hasScimToken && !lastToken && (
            <span className="text-xs text-gray-600 dark:text-gray-400">A SCIM token is already configured.</span>
          )}
        </div>
        {showToken && lastToken && (
          <div className="mt-4 p-3 rounded-md bg-gray-50 dark:bg-gray-800 border border-gray-200 dark:border-gray-700">
            <p className="text-xs font-semibold text-gray-700 dark:text-gray-200 mb-1">New SCIM token</p>
            <p className="text-xs text-gray-900 dark:text-gray-100 break-all">{lastToken}</p>
            <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
              Copy and store this token securely. You will not be able to see it again.
            </p>
          </div>
        )}
      </div>
    </div>
  );
}

