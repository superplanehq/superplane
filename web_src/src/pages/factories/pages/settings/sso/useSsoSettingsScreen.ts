import { useEffect, useState } from "react";
import { useSearchParams } from "react-router";

import type { SsoPutConnectionRequest } from "@/api-client/types.gen";
import { usePermissions } from "@/contexts/usePermissions";
import { getApiErrorMessage } from "@/lib/errors";
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

import type { SsoSettingsActions, SsoSettingsModel } from "./ssoSettingsTypes";

export function useSsoSettingsScreen(organizationId: string) {
  const data = useSsoSettingsData(organizationId);
  const feedback = useSsoFeedback();
  const model = ssoModel(data, feedback);
  const actions = ssoActions(data, feedback);
  return { loading: data.connectionQuery.isLoading, model, actions };
}

function useSsoSettingsData(organizationId: string) {
  const { data: organization } = useOrganization(organizationId);
  const { canAct } = usePermissions();
  return {
    organization,
    canUpdate: canAct("sso", "update"),
    connectionQuery: useSsoConnection(organizationId),
    domainsQuery: useSsoDomains(organizationId),
    tokensQuery: useSsoScimTokens(organizationId),
    save: useSaveSsoConnection(organizationId),
    activate: useActivateSsoConnection(organizationId),
    disable: useDisableSsoConnection(organizationId),
    remove: useDeleteSsoConnection(organizationId),
    createDomain: useCreateSsoDomain(organizationId),
    verifyDomain: useVerifySsoDomain(organizationId),
    deleteDomain: useDeleteSsoDomain(organizationId),
    createToken: useCreateSsoScimToken(organizationId),
    revokeToken: useRevokeSsoScimToken(organizationId),
  };
}

function useSsoFeedback() {
  const [revealedToken, setRevealedToken] = useState<string | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);
  const [secretEpoch, setSecretEpoch] = useState(0);
  const [params, setParams] = useSearchParams();
  useEffect(() => {
    readTestFlag(params, setNotice, setActionError, setParams);
  }, [params, setParams]);
  return { revealedToken, setRevealedToken, actionError, setActionError, notice, secretEpoch, setSecretEpoch };
}

function readTestFlag(
  params: URLSearchParams,
  setNotice: (value: string | null) => void,
  setActionError: (value: string | null) => void,
  setParams: (next: URLSearchParams, options: { replace: boolean }) => void,
) {
  const flag = params.get("sso_test");
  if (!flag) {
    return;
  }
  if (flag === "ok") {
    setNotice("The sign-in test succeeded.");
  } else {
    setActionError("The sign-in test failed.");
  }
  const next = new URLSearchParams(params);
  next.delete("sso_test");
  setParams(next, { replace: true });
}

type SettingsData = ReturnType<typeof useSsoSettingsData>;
type SettingsFeedback = ReturnType<typeof useSsoFeedback>;

function ssoModel(data: SettingsData, feedback: SettingsFeedback): SsoSettingsModel {
  const connection = data.connectionQuery.data?.exists ? data.connectionQuery.data.connection : null;
  const loadError = data.connectionQuery.error ?? data.domainsQuery.error ?? data.tokensQuery.error;
  const busy = [
    data.save,
    data.activate,
    data.disable,
    data.remove,
    data.createDomain,
    data.verifyDomain,
    data.deleteDomain,
    data.createToken,
    data.revokeToken,
  ].some((mutation) => mutation.isPending);
  return {
    licensed: hasOrganizationEnterpriseFeature(data.organization, "sso"),
    canUpdate: data.canUpdate,
    busy,
    secretEpoch: feedback.secretEpoch,
    connection,
    domains: data.domainsQuery.data ?? [],
    tokens: data.tokensQuery.data ?? [],
    error:
      feedback.actionError ?? (loadError ? getApiErrorMessage(loadError, "Single Sign-On could not be loaded.") : null),
    notice: feedback.notice,
    revealedToken: feedback.revealedToken,
  };
}

function ssoActions(data: SettingsData, feedback: SettingsFeedback): SsoSettingsActions {
  const fail = (mutationError: unknown) => {
    feedback.setActionError(getApiErrorMessage(mutationError, "The request failed. Try again."));
  };
  const start = () => feedback.setActionError(null);
  const connection = data.connectionQuery.data?.exists ? data.connectionQuery.data.connection : null;
  return {
    onAddDomain: (domain) => {
      start();
      data.createDomain.mutate(domain, { onError: fail });
    },
    onVerifyDomain: (id) => {
      start();
      data.verifyDomain.mutate(id, { onError: fail });
    },
    onDeleteDomain: (id) => {
      start();
      data.deleteDomain.mutate(id, { onError: fail });
    },
    onSave: (input: SsoPutConnectionRequest) => {
      start();
      data.save.mutate(input, {
        onError: fail,
        onSuccess: () => feedback.setSecretEpoch((current) => current + 1),
      });
    },
    onTest: () => {
      if (connection?.id) {
        window.location.href = `/auth/sso/${connection.id}/start?test=1`;
      }
    },
    onActivate: () => {
      start();
      data.activate.mutate(undefined, { onError: fail });
    },
    onDisable: () => {
      start();
      data.disable.mutate(undefined, { onError: fail });
    },
    onDelete: () => {
      start();
      data.remove.mutate(undefined, { onError: fail });
    },
    onCreateToken: (description) => {
      start();
      data.createToken.mutate(description, {
        onError: fail,
        onSuccess: (response) => feedback.setRevealedToken(response.data?.secret ?? null),
      });
    },
    onRevokeToken: (id) => {
      start();
      data.revokeToken.mutate(id, { onError: fail });
    },
    onDismissToken: () => feedback.setRevealedToken(null),
  };
}
