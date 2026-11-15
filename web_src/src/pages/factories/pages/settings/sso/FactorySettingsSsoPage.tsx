import { useState } from "react";
import { useSearchParams } from "react-router";

import { hasOrganizationEnterpriseFeature } from "@/lib/license";
import { useOrganization } from "@/hooks/useOrganizationData";
import {
  useActivateSsoConnection,
  useCreateSsoDomain,
  useCreateSsoScimToken,
  useDeleteSsoConnection,
  useDeleteSsoDomain,
  useDisableSsoConnection,
  useSaveSsoConnection,
  useSsoConnection,
  useSsoDomains,
  useSsoScimTokens,
  useRevokeSsoScimToken,
  useVerifySsoDomain,
} from "@/hooks/useSso";

import { FactorySettingsPageFrame } from "../FactorySettingsCard";
import { useFactorySettingsLayout } from "../factorySettingsLayoutContext";
import { SsoSettingsView } from "./SsoSettingsView";

export function FactorySettingsSsoPage() {
  const { organizationId } = useFactorySettingsLayout();
  const { data: organization } = useOrganization(organizationId);
  const connectionQuery = useSsoConnection(organizationId);
  const domainsQuery = useSsoDomains(organizationId);
  const tokensQuery = useSsoScimTokens(organizationId);
  const save = useSaveSsoConnection(organizationId);
  const activate = useActivateSsoConnection(organizationId);
  const disable = useDisableSsoConnection(organizationId);
  const remove = useDeleteSsoConnection(organizationId);
  const createDomain = useCreateSsoDomain(organizationId);
  const verifyDomain = useVerifySsoDomain(organizationId);
  const deleteDomain = useDeleteSsoDomain(organizationId);
  const createToken = useCreateSsoScimToken(organizationId);
  const revokeToken = useRevokeSsoScimToken(organizationId);
  const [revealedToken, setRevealedToken] = useState<string | null>(null);
  const [params] = useSearchParams();
  const testState = params.get("sso_test");

  const connection = connectionQuery.data?.exists ? connectionQuery.data.connection : null;
  const licensed = hasOrganizationEnterpriseFeature(organization, "sso");
  const error =
    connectionQuery.error instanceof Error
      ? connectionQuery.error.message
      : testState === "failed"
        ? "The sign-in test failed."
        : null;

  return (
    <FactorySettingsPageFrame title="Single Sign-On" subtitle="Connect one identity provider for this organization.">
      {connectionQuery.isLoading ? (
        <p>Loading Single Sign-On.</p>
      ) : (
        <SsoSettingsView
          key={`${connection?.id ?? "new"}-${connection?.status ?? ""}-${connection?.lastTestedAt ?? ""}`}
          model={{
            licensed,
            connection,
            domains: domainsQuery.data ?? [],
            tokens: tokensQuery.data ?? [],
            error,
            revealedToken,
          }}
          actions={{
            onAddDomain: (domain) => createDomain.mutate(domain),
            onVerifyDomain: (id) => verifyDomain.mutate(id),
            onDeleteDomain: (id) => deleteDomain.mutate(id),
            onSave: (input) => save.mutate(input),
            onTest: () => {
              if (connection?.id) {
                window.location.href = `/auth/sso/${connection.id}/start?test=1`;
              }
            },
            onActivate: () => activate.mutate(),
            onDisable: () => disable.mutate(),
            onDelete: () => remove.mutate(),
            onCreateToken: (description) =>
              createToken.mutate(description, {
                onSuccess: (response) => setRevealedToken(response.data?.secret ?? null),
              }),
            onRevokeToken: (id) => revokeToken.mutate(id),
            onDismissToken: () => setRevealedToken(null),
          }}
        />
      )}
    </FactorySettingsPageFrame>
  );
}
