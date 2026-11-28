import { useEffect, useState } from "react";
import { useSearchParams } from "react-router";

import type { SsoPutConnectionRequest } from "@/api-client/types.gen";
import { usePermissions } from "@/contexts/usePermissions";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
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

import { ssoTestFailureMessage, type SsoSettingsActions, type SsoSettingsModel } from "./ssoSettingsTypes";

export function useSsoSettingsScreen(organizationId: string) {
  const data = useSsoSettingsData(organizationId);
  const feedback = useSsoFeedback();
  const loadError = data.connectionQuery.error ?? data.domainsQuery.error ?? data.tokensQuery.error;
  useEffect(() => {
    if (loadError) {
      showErrorToast(getApiErrorMessage(loadError, "Single Sign-On could not be loaded."));
    }
  }, [loadError]);
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
  const [domainError, setDomainError] = useState<{ id: string; message: string } | null>(null);
  const [testFailure, setTestFailure] = useState<string | null>(null);
  const [secretEpoch, setSecretEpoch] = useState(0);
  const [params, setParams] = useSearchParams();
  useEffect(() => {
    applyTestFlag(params, setParams, setTestFailure);
  }, [params, setParams]);
  return { revealedToken, setRevealedToken, domainError, setDomainError, testFailure, secretEpoch, setSecretEpoch };
}

function applyTestFlag(
  params: URLSearchParams,
  setParams: (next: URLSearchParams, options: { replace: boolean }) => void,
  setTestFailure: (message: string | null) => void,
) {
  const flag = params.get("sso_test");
  if (!flag) {
    return;
  }
  if (flag === "ok") {
    showSuccessToast("The sign-in test succeeded.");
    setTestFailure(null);
  } else {
    const message = ssoTestFailureMessage(params.get("reason"));
    showErrorToast(message);
    setTestFailure(message);
  }
  const next = new URLSearchParams(params);
  next.delete("sso_test");
  next.delete("reason");
  setParams(next, { replace: true });
}

type SettingsData = ReturnType<typeof useSsoSettingsData>;
type SettingsFeedback = ReturnType<typeof useSsoFeedback>;

function ssoModel(data: SettingsData, feedback: SettingsFeedback): SsoSettingsModel {
  const connection = data.connectionQuery.data?.exists ? data.connectionQuery.data.connection : null;
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
    domainError: feedback.domainError,
    revealedToken: feedback.revealedToken,
    testFailure: feedback.testFailure,
  };
}

function ssoActions(data: SettingsData, feedback: SettingsFeedback): SsoSettingsActions {
  const fail = (mutationError: unknown) => {
    showErrorToast(getApiErrorMessage(mutationError, "The request failed. Try again."));
  };
  const start = () => feedback.setDomainError(null);
  const connection = data.connectionQuery.data?.exists ? data.connectionQuery.data.connection : null;
  return {
    onAddDomain: (domain) => {
      start();
      data.createDomain.mutate(domain, { onError: fail, onSuccess: () => showSuccessToast("Domain added.") });
    },
    onVerifyDomain: (id) => {
      start();
      data.verifyDomain.mutate(id, {
        onError: (error) =>
          feedback.setDomainError({ id, message: getApiErrorMessage(error, "The domain could not be verified.") }),
        onSuccess: () => showSuccessToast("Domain verified."),
      });
    },
    onDeleteDomain: (id) => {
      start();
      data.deleteDomain.mutate(id, { onError: fail, onSuccess: () => showSuccessToast("Domain deleted.") });
    },
    onSave: (input: SsoPutConnectionRequest) => {
      start();
      data.save.mutate(input, {
        onError: fail,
        onSuccess: () => {
          if (input.clientSecret) {
            feedback.setSecretEpoch((current) => current + 1);
          }
          showSuccessToast("Connection saved.");
        },
      });
    },
    onTest: () => {
      if (connection?.id) {
        const redirect = encodeURIComponent(window.location.pathname);
        window.location.href = `/auth/sso/${connection.id}/start?test=1&redirect=${redirect}`;
      }
    },
    onActivate: () => {
      start();
      data.activate.mutate(undefined, {
        onError: fail,
        onSuccess: () => showSuccessToast("Single Sign-On is active."),
      });
    },
    onDisable: () => {
      start();
      data.disable.mutate(undefined, {
        onError: fail,
        onSuccess: () => showSuccessToast("Single Sign-On is disabled."),
      });
    },
    onDelete: () => {
      start();
      data.remove.mutate(undefined, { onError: fail, onSuccess: () => showSuccessToast("Connection deleted.") });
    },
    onCreateToken: (description) => {
      start();
      data.createToken.mutate(description, {
        onError: fail,
        onSuccess: (response) => {
          feedback.setRevealedToken(response.data?.secret ?? null);
          showSuccessToast("SCIM token created.");
        },
      });
    },
    onRevokeToken: (id) => {
      start();
      data.revokeToken.mutate(id, { onError: fail, onSuccess: () => showSuccessToast("SCIM token revoked.") });
    },
    onDismissToken: () => feedback.setRevealedToken(null),
  };
}
