import { useEffect, useRef, useState, type Dispatch, type SetStateAction } from "react";

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
  const previousConnection = useRef(connection);
  const [confirm, setConfirm] = useState<ConfirmRequest | null>(null);
  const [requiredConfirm, setRequiredConfirm] = useState(false);
  const [sessionError, setSessionError] = useState<string | null>(null);
  const access = accessState(model);
  const set = formSetters(setForm);
  const permitted = model.canUpdate !== false;
  const allowed = permitted && !model.busy;
  const ask = (title: string, body: string, actionLabel: string, run: () => void) =>
    setConfirm({ title, body, actionLabel, run });
  const requestRequired = () => setRequiredConfirm(true);
  const createConnection = () => {
    const seconds = sessionAgeSeconds(form.sessionHours) ?? 86400;
    actions.onSave(
      ssoConnectionInput({ ...form, issuer: "", clientId: "", clientSecret: "" }, form.enforcement, seconds),
    );
  };
  const saveIdentity = () => saveCard(actions, connection, form, "identity");
  const saveCredentials = () => saveCard(actions, connection, form, "credentials");
  const saveProvisioning = () => saveCard(actions, connection, form, "provisioning");
  const savePolicy = () => {
    const seconds = sessionAgeSeconds(form.sessionHours);
    if (seconds == null) {
      setSessionError("Enter 15 minutes to 7 days, as hours or minutes.");
      return;
    }
    setSessionError(null);
    saveCard(actions, connection, form, "policy", seconds);
  };
  const resetIdentity = () => {
    const base = initialForm(connection);
    setForm((current) => ({ ...current, displayName: base.displayName, preset: base.preset }));
  };
  const resetCredentials = () => {
    const base = initialForm(connection);
    setForm((current) => ({
      ...current,
      issuer: base.issuer,
      clientId: base.clientId,
      clientSecret: "",
      ...claimFields(connection),
    }));
  };
  const resetProvisioning = () => {
    const base = initialForm(connection);
    setForm((current) => ({ ...current, jitEnabled: base.jitEnabled, defaultRole: base.defaultRole }));
  };
  const resetPolicy = () => {
    const base = initialForm(connection);
    setSessionError(null);
    setForm((current) => ({
      ...current,
      enforcement: base.enforcement,
      ownerBypass: base.ownerBypass,
      sessionHours: base.sessionHours,
      idpLogout: base.idpLogout,
    }));
  };
  useEffect(() => {
    const previous = previousConnection.current;
    previousConnection.current = connection;
    setForm((current) => refreshForm(current, previous, connection));
  }, [connection]);
  useEffect(() => {
    if (!model.secretEpoch) {
      return;
    }
    setForm((current) => ({ ...current, clientSecret: "" }));
  }, [model.secretEpoch]);
  const draft: SsoDraft = {
    canChange: Boolean(model.licensed) && allowed,
    canUpdate: allowed,
    permitted,
    identityDirty: isIdentityDirty(form, connection),
    credentialsDirty: isCredentialsDirty(form, connection),
    provisioningDirty: isProvisioningDirty(form, connection),
    policyDirty: isPolicyDirty(form, connection),
    providerReady: Boolean(connection?.issuer?.trim() && connection.clientId?.trim() && connection.clientSecretSet),
    sessionError,
    ...access,
    ...form,
    ...set,
    ask,
    requestRequired,
    createConnection,
    saveIdentity,
    saveCredentials,
    resetIdentity,
    resetCredentials,
    resetProvisioning,
    resetPolicy,
    saveProvisioning,
    savePolicy,
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

function refreshForm(current: SsoForm, previous?: SsoConnection | null, connection?: SsoConnection | null): SsoForm {
  if (previous?.id !== connection?.id) {
    return initialForm(connection);
  }
  const base = initialForm(previous);
  const saved = initialForm(connection);
  let next = { ...current, ...claimFields(connection) };
  const fields = { ...textFields(connection), ...flagFields(connection) };
  for (const key of Object.keys(fields) as (keyof typeof fields)[]) {
    // Compare durations by value so an unchanged "15m" also follows a refresh.
    const unchanged =
      key === "sessionHours"
        ? sessionAgeSeconds(current[key]) === sessionAgeSeconds(base[key])
        : current[key] === base[key];
    if (unchanged) {
      next = { ...next, [key]: saved[key] };
    }
  }
  return next;
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

type CardKind = "identity" | "credentials" | "provisioning" | "policy";

function saveCard(
  actions: SsoSettingsActions,
  connection: SsoConnection | null | undefined,
  form: SsoForm,
  kind: CardKind,
  policySeconds?: number,
) {
  const base = initialForm(connection);
  const seconds = policySeconds ?? connection?.maxSessionAgeSeconds ?? 86400;
  const next = { ...base };
  if (kind === "identity") {
    next.displayName = form.displayName;
    next.preset = form.preset;
  }
  if (kind === "credentials") {
    next.issuer = form.issuer;
    next.clientId = form.clientId;
    next.clientSecret = form.clientSecret;
    next.scopes = form.scopes;
    next.subjectClaim = form.subjectClaim;
    next.emailClaim = form.emailClaim;
    next.nameClaim = form.nameClaim;
  }
  if (kind === "provisioning") {
    next.jitEnabled = form.jitEnabled;
    next.defaultRole = form.defaultRole;
  }
  if (kind === "policy") {
    next.ownerBypass = form.ownerBypass;
    next.idpLogout = form.idpLogout;
  }
  const enforcement = kind === "policy" ? form.enforcement : base.enforcement;
  actions.onSave(ssoConnectionInput(next, enforcement, seconds));
}

function isIdentityDirty(form: SsoForm, connection?: SsoConnection | null) {
  const base = initialForm(connection);
  return form.preset !== base.preset || form.displayName !== base.displayName;
}

function isCredentialsDirty(form: SsoForm, connection?: SsoConnection | null) {
  const base = initialForm(connection);
  return form.issuer !== base.issuer || form.clientId !== base.clientId || form.clientSecret !== "";
}

function isProvisioningDirty(form: SsoForm, connection?: SsoConnection | null) {
  const base = initialForm(connection);
  return form.jitEnabled !== base.jitEnabled || form.defaultRole !== base.defaultRole;
}

function isPolicyDirty(form: SsoForm, connection?: SsoConnection | null) {
  const base = initialForm(connection);
  const seconds = sessionAgeSeconds(form.sessionHours);
  const saved = connection?.maxSessionAgeSeconds ?? 86400;
  return (
    form.enforcement !== base.enforcement ||
    form.ownerBypass !== base.ownerBypass ||
    form.idpLogout !== base.idpLogout ||
    seconds == null ||
    seconds !== saved
  );
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
