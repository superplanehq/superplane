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
import { useOktaSSO } from "./useOktaSSO";

function copyToClipboard(text: string) {
  navigator.clipboard.writeText(text).catch(() => {});
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
  return <span className={`text-sm ${message.includes("Failed") ? "text-red-600" : "text-green-600"}`}>{message}</span>;
}

function CopyableField({ label, hint, value }: { label: string; hint: string; value: string }) {
  return (
    <Field className="space-y-1.5">
      <Label className="text-sm font-medium text-gray-700 dark:text-gray-300">{label}</Label>
      <p className="text-xs text-gray-500 dark:text-gray-400">{hint}</p>
      <div className="flex items-center gap-2 max-w-sm">
        <Input type="text" value={value} readOnly className="bg-gray-50 dark:bg-gray-700 text-gray-600" />
        <Button type="button" variant="outline" onClick={() => copyToClipboard(value)}>
          Copy
        </Button>
      </div>
    </Field>
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

type SAMLSectionProps = {
  samlIdpSSOURL: string;
  setSamlIdpSSOURL: (v: string) => void;
  samlIdpIssuer: string;
  setSamlIdpIssuer: (v: string) => void;
  samlIdpCertificatePEM: string;
  setSamlIdpCertificatePEM: (v: string) => void;
  samlEnabled: boolean;
  setSamlEnabled: (v: boolean) => void;
  samlSaving: boolean;
  samlMessage: string | null;
  certConfigured: boolean;
  acsURL: string;
  entityID: string;
  loginURL: string;
  canUpdate: boolean;
  permissionsLoading: boolean;
  onSave: () => void;
};

function SAMLSection({
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
  certConfigured,
  acsURL,
  entityID,
  loginURL,
  canUpdate,
  permissionsLoading,
  onSave,
}: SAMLSectionProps) {
  return (
    <Fieldset className="bg-white dark:bg-gray-800 rounded-lg border border-gray-300 dark:border-gray-800 p-6 space-y-5">
      <div>
        <p className="text-sm font-semibold text-gray-800 dark:text-white">SAML Sign-In</p>
        <p className="text-sm text-gray-500 dark:text-gray-400 mt-0.5">
          Allow members to log in via Okta SAML 2.0. Paste the values from Okta&apos;s{" "}
          <em>View SAML setup instructions</em> panel, then copy the ACS URL and SP Entity ID into your Okta app&apos;s
          SAML settings.
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
          disabled={!canUpdate}
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
          disabled={!canUpdate}
          className="max-w-sm"
        />
      </Field>
      <CertificateField
        configured={certConfigured}
        disabled={!canUpdate}
        value={samlIdpCertificatePEM}
        onChange={setSamlIdpCertificatePEM}
      />
      <CopyableField
        label="ACS URL (Single Sign-On URL)"
        hint="Paste this into your Okta app's Single sign on URL field."
        value={acsURL}
      />
      <CopyableField
        label="SP Entity ID (Audience URI)"
        hint="Paste this into your Okta app's Audience URI (SP Entity ID) field."
        value={entityID}
      />
      <CopyableField
        label="Employee login URL"
        hint="Share this URL with employees to let them sign in via Okta."
        value={loginURL}
      />
      <Field>
        <div className="flex items-center gap-3">
          <Switch
            checked={samlEnabled}
            onCheckedChange={setSamlEnabled}
            disabled={!canUpdate}
            aria-label="Enable SAML sign-in"
          />
          <Label className="text-sm font-medium text-gray-700 dark:text-gray-300">Enable SAML sign-in</Label>
        </div>
      </Field>
      <div className="flex items-center gap-4 pt-1">
        <PermissionTooltip
          allowed={canUpdate || permissionsLoading}
          message="You don't have permission to update SSO settings."
        >
          <LoadingButton
            type="button"
            onClick={onSave}
            disabled={!canUpdate}
            loading={samlSaving}
            loadingText="Saving..."
          >
            Save
          </LoadingButton>
        </PermissionTooltip>
        <StatusMessage message={samlMessage} />
      </div>
    </Fieldset>
  );
}

function SCIMSection({
  scimEnabled,
  setScimEnabled,
  scimSaving,
  scimMessage,
  rotatingToken,
  tokenConfigured,
  scimBaseURL,
  canUpdate,
  permissionsLoading,
  onSave,
  onRotateToken,
}: {
  scimEnabled: boolean;
  setScimEnabled: (v: boolean) => void;
  scimSaving: boolean;
  scimMessage: string | null;
  rotatingToken: boolean;
  tokenConfigured: boolean;
  scimBaseURL: string;
  canUpdate: boolean;
  permissionsLoading: boolean;
  onSave: () => void;
  onRotateToken: () => void;
}) {
  return (
    <Fieldset className="bg-white dark:bg-gray-800 rounded-lg border border-gray-300 dark:border-gray-800 p-6 space-y-5">
      <div>
        <p className="text-sm font-semibold text-gray-800 dark:text-white">SCIM Provisioning</p>
        <p className="text-sm text-gray-500 dark:text-gray-400 mt-0.5">
          Automatically provision and deprovision members via Okta. Configure SCIM in your Okta app&apos;s{" "}
          <em>Provisioning</em> tab.
        </p>
      </div>
      <CopyableField
        label="SCIM base URL"
        hint="Paste this into your Okta SCIM connector base URL field."
        value={scimBaseURL}
      />
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
          allowed={canUpdate || permissionsLoading}
          message="You don't have permission to update SSO settings."
        >
          <LoadingButton
            type="button"
            variant="outline"
            onClick={onRotateToken}
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
          <Label className="text-sm font-medium text-gray-700 dark:text-gray-300">Enable SCIM provisioning</Label>
        </div>
      </Field>
      <div className="flex items-center gap-4 pt-1">
        <PermissionTooltip
          allowed={canUpdate || permissionsLoading}
          message="You don't have permission to update SSO settings."
        >
          <LoadingButton
            type="button"
            onClick={onSave}
            disabled={!canUpdate}
            loading={scimSaving}
            loadingText="Saving..."
          >
            Save
          </LoadingButton>
        </PermissionTooltip>
        <StatusMessage message={scimMessage} />
      </div>
    </Fieldset>
  );
}

export function OktaSSO() {
  const { organizationId } = useParams<{ organizationId: string }>();
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  const publicBaseURL = usePublicBaseURL();
  usePageTitle(["SSO"]);

  const canUpdate = canAct("org", "update");
  const { acsURL, entityID, loginURL, scimBaseURL } = getOktaURLs(publicBaseURL, organizationId ?? "");
  const state = useOktaSSO(organizationId, canUpdate);

  const { scimBearerTokenConfigured: tokenConfigured = false, samlIdpCertificateConfigured: certConfigured = false } =
    state.settings ?? {};

  if (state.loadError) {
    return <p className="text-sm text-red-500 pt-6">{state.loadError}</p>;
  }

  return (
    <div className="space-y-6 pt-6 text-left">
      <SAMLSection
        samlIdpSSOURL={state.samlIdpSSOURL}
        setSamlIdpSSOURL={state.setSamlIdpSSOURL}
        samlIdpIssuer={state.samlIdpIssuer}
        setSamlIdpIssuer={state.setSamlIdpIssuer}
        samlIdpCertificatePEM={state.samlIdpCertificatePEM}
        setSamlIdpCertificatePEM={state.setSamlIdpCertificatePEM}
        samlEnabled={state.samlEnabled}
        setSamlEnabled={state.setSamlEnabled}
        samlSaving={state.samlSaving}
        samlMessage={state.samlMessage}
        certConfigured={certConfigured}
        acsURL={acsURL}
        entityID={entityID}
        loginURL={loginURL}
        canUpdate={canUpdate}
        permissionsLoading={permissionsLoading}
        onSave={state.handleSamlSave}
      />
      <SCIMSection
        scimEnabled={state.scimEnabled}
        setScimEnabled={state.setScimEnabled}
        scimSaving={state.scimSaving}
        scimMessage={state.scimMessage}
        rotatingToken={state.rotatingToken}
        tokenConfigured={tokenConfigured}
        scimBaseURL={scimBaseURL}
        canUpdate={canUpdate}
        permissionsLoading={permissionsLoading}
        onSave={state.handleScimSave}
        onRotateToken={state.handleRotateToken}
      />
      <Dialog
        open={!!state.newToken}
        onOpenChange={(open) => {
          if (!open) state.setNewToken(null);
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
              value={state.newToken || ""}
              readOnly
              className="font-mono text-xs bg-gray-50 dark:bg-gray-700"
            />
          </div>
          <DialogFooter>
            <Button
              type="button"
              onClick={() => {
                if (state.newToken) copyToClipboard(state.newToken);
                state.setNewToken(null);
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
