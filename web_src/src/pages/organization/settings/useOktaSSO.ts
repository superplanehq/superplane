import { useEffect, useState } from "react";
import type { OrganizationsOktaIdpSettings } from "../../../api-client/types.gen";
import {
  organizationsGetOktaIdpSettings,
  organizationsRotateOktaScimBearerToken,
  organizationsUpdateOktaIdpSettings,
} from "../../../api-client/sdk.gen";
import { withOrganizationHeader } from "../../../lib/withOrganizationHeader";

function flashMessage(setter: (msg: string | null) => void, message: string, duration = 3000) {
  setter(message);
  setTimeout(() => setter(null), duration);
}

export function useOktaSSO(organizationId: string | undefined, canUpdate: boolean) {
  const [settings, setSettings] = useState<OrganizationsOktaIdpSettings | null>(null);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [samlIdpSSOURL, setSamlIdpSSOURL] = useState("");
  const [samlIdpIssuer, setSamlIdpIssuer] = useState("");
  const [samlIdpCertificatePEM, setSamlIdpCertificatePEM] = useState("");
  const [samlEnabled, setSamlEnabled] = useState(true);
  const [samlSaving, setSamlSaving] = useState(false);
  const [samlMessage, setSamlMessage] = useState<string | null>(null);
  const [scimEnabled, setScimEnabled] = useState(true);
  const [scimSaving, setScimSaving] = useState(false);
  const [scimMessage, setScimMessage] = useState<string | null>(null);
  const [rotatingToken, setRotatingToken] = useState(false);
  const [newToken, setNewToken] = useState<string | null>(null);

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
    if (!canUpdate || !organizationId) return;
    setSamlSaving(true);
    try {
      const cert = samlIdpCertificatePEM.trim();
      const body = {
        samlIdpSsoUrl: samlIdpSSOURL.trim(),
        samlIdpIssuer: samlIdpIssuer.trim(),
        samlEnabled,
        ...(cert ? { samlIdpCertificatePem: cert } : {}),
      };
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

  return {
    settings,
    loadError,
    samlIdpSSOURL,
    setSamlIdpSSOURL,
    samlIdpIssuer,
    setSamlIdpIssuer,
    samlIdpCertificatePEM,
    setSamlIdpCertificatePEM,
    samlEnabled,
    setSamlEnabled,
    samlSaving,
    samlMessage,
    scimEnabled,
    setScimEnabled,
    scimSaving,
    scimMessage,
    rotatingToken,
    newToken,
    setNewToken,
    handleSamlSave,
    handleScimSave,
    handleRotateToken,
  };
}
