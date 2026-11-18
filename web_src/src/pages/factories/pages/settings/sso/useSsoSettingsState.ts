import { useEffect, useState, type Dispatch, type SetStateAction } from "react";

import type { SsoConnection, SsoPutConnectionRequest } from "@/api-client/types.gen";

import type { ConfirmRequest, SsoDraft, SsoSettingsActions, SsoSettingsModel } from "./ssoSettingsTypes";

export type SsoForm = {
  domain: string;
  displayName: string;
  preset: string;
  issuer: string;
  clientId: string;
  clientSecret: string;
  jitEnabled: boolean;
  defaultRole: string;
  enforcement: string;
  ownerBypass: boolean;
  sessionHours: string;
  idpLogout: boolean;
  tokenDescription: string;
  scopes: string[];
  subjectClaim: string;
  emailClaim: string;
  nameClaim: string;
  loadedPreset: string;
};

export function useSsoSettingsState(model: SsoSettingsModel, actions: SsoSettingsActions) {
  const connection = model.connection;
  const [form, setForm] = useState(() => initialForm(connection));
  const [confirm, setConfirm] = useState<ConfirmRequest | null>(null);
  const [requiredConfirm, setRequiredConfirm] = useState(false);
  const [sessionError, setSessionError] = useState<string | null>(null);
  const access = accessState(model);
  const set = formSetters(setForm);
  const allowed = model.canUpdate !== false && !model.busy;
  const ask = (title: string, body: string, actionLabel: string, run: () => void) =>
    setConfirm({ title, body, actionLabel, run });
  const requestRequired = () => setRequiredConfirm(true);
  const save = (nextEnforcement = form.enforcement) => {
    const seconds = sessionAgeSeconds(form.sessionHours);
    if (seconds == null) {
      setSessionError("Enter 15 minutes to 7 days, as hours or minutes.");
      return;
    }
    setSessionError(null);
    actions.onSave(ssoConnectionInput(form, nextEnforcement, seconds));
  };
  useEffect(() => {
    if (!model.secretEpoch) {
      return;
    }
    setForm((current) => ({ ...current, clientSecret: "" }));
  }, [model.secretEpoch]);
  const draft: SsoDraft = {
    canChange: Boolean(model.licensed) && allowed,
    canUpdate: allowed,
    sessionError,
    ...access,
    ...form,
    ...set,
    ask,
    requestRequired,
    save,
  };
  return { exists: access.exists, draft, confirm, setConfirm, requiredConfirm, setRequiredConfirm };
}

function textFields(connection?: SsoConnection | null) {
  return {
    displayName: connection?.displayName || "Okta",
    preset: connection?.preset || "PRESET_OKTA",
    issuer: connection?.issuer || "",
    clientId: connection?.clientId || "",
    defaultRole: connection?.defaultRole || "org_operator",
  };
}

function flagFields(connection?: SsoConnection | null) {
  const seconds = connection?.maxSessionAgeSeconds ?? 86400;
  return {
    jitEnabled: connection?.jitEnabled !== false,
    enforcement: connection?.enforcement || "ENFORCEMENT_OPTIONAL",
    ownerBypass: connection?.allowOwnerBypass !== false,
    sessionHours: formatSessionAge(seconds),
    idpLogout: Boolean(connection?.idpLogoutEnabled),
  };
}

function initialForm(connection?: SsoConnection | null): SsoForm {
  return {
    domain: "",
    clientSecret: "",
    tokenDescription: "Okta",
    ...textFields(connection),
    ...flagFields(connection),
    ...claimFields(connection),
  };
}

function claimFields(connection?: SsoConnection | null) {
  return {
    scopes: connection?.scopes ?? [],
    subjectClaim: connection?.subjectClaim ?? "",
    emailClaim: connection?.emailClaim ?? "",
    nameClaim: connection?.nameClaim ?? "",
    loadedPreset: connection?.preset || "PRESET_OKTA",
  };
}

function accessState(model: SsoSettingsModel) {
  return {
    exists: Boolean(model.connection?.id),
    tested: model.connection?.lastTest?.ok === true,
    verified: model.domains.some((item) => Boolean(item.verifiedAt)),
  };
}

function formSetters(setForm: Dispatch<SetStateAction<SsoForm>>) {
  return {
    setDomain: updateForm(setForm, "domain"),
    setDisplayName: updateForm(setForm, "displayName"),
    setPreset: updateForm(setForm, "preset"),
    setIssuer: updateForm(setForm, "issuer"),
    setClientId: updateForm(setForm, "clientId"),
    setClientSecret: updateForm(setForm, "clientSecret"),
    setJitEnabled: updateForm(setForm, "jitEnabled"),
    setDefaultRole: updateForm(setForm, "defaultRole"),
    setEnforcement: updateForm(setForm, "enforcement"),
    setOwnerBypass: updateForm(setForm, "ownerBypass"),
    setSessionHours: updateForm(setForm, "sessionHours"),
    setIdpLogout: updateForm(setForm, "idpLogout"),
    setTokenDescription: updateForm(setForm, "tokenDescription"),
  };
}

function updateForm<K extends keyof SsoForm>(setForm: Dispatch<SetStateAction<SsoForm>>, key: K) {
  return (value: SsoForm[K]) => setForm((current) => ({ ...current, [key]: value }));
}

export function sessionAgeSeconds(value: string): number | null {
  const trimmed = value.trim();
  const minutes = /^(\d+(?:\.\d+)?)m$/.exec(trimmed);
  const hours = /^(\d+(?:\.\d+)?)$/.exec(trimmed);
  const amount = minutes?.[1] ?? hours?.[1];
  if (!amount) {
    return null;
  }
  const seconds = Math.round(Number(amount) * (minutes ? 60 : 3600));
  if (!Number.isFinite(seconds) || seconds < 900 || seconds > 604800) {
    return null;
  }
  return seconds;
}

export function formatSessionAge(seconds: number): string {
  if (seconds % 3600 === 0) {
    return String(seconds / 3600);
  }
  for (let digits = 1; digits <= 2; digits += 1) {
    const text = trimDecimal((seconds / 3600).toFixed(digits));
    if (sessionAgeSeconds(text) === seconds) {
      return text;
    }
  }
  if (seconds % 60 === 0) {
    return `${seconds / 60}m`;
  }
  for (let digits = 3; digits <= 8; digits += 1) {
    const text = trimDecimal((seconds / 3600).toFixed(digits));
    if (sessionAgeSeconds(text) === seconds) {
      return text;
    }
  }
  return String(seconds / 3600);
}

function trimDecimal(value: string): string {
  return value.replace(/(\.\d*?)0+$/, "$1").replace(/\.$/, "");
}

export function ssoConnectionInput(
  form: SsoForm,
  enforcement: string,
  maxSessionAgeSeconds: number,
): SsoPutConnectionRequest {
  const presetChanged = form.preset !== form.loadedPreset;
  return {
    preset: form.preset as SsoPutConnectionRequest["preset"],
    displayName: form.displayName,
    issuer: form.issuer,
    clientId: form.clientId,
    clientSecret: form.clientSecret,
    scopes: presetChanged ? [] : form.scopes,
    subjectClaim: presetChanged ? "" : form.subjectClaim,
    emailClaim: presetChanged ? "" : form.emailClaim,
    nameClaim: presetChanged ? "" : form.nameClaim,
    jitEnabled: form.jitEnabled,
    defaultRole: form.defaultRole,
    enforcement: enforcement as SsoPutConnectionRequest["enforcement"],
    allowOwnerBypass: form.ownerBypass,
    maxSessionAgeSeconds,
    idpLogoutEnabled: form.idpLogout,
  };
}
