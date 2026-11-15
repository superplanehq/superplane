import { useState, type Dispatch, type SetStateAction } from "react";

import type { SsoConnection, SsoPutConnectionRequest } from "@/api-client/types.gen";

import type { ConfirmRequest, SsoDraft, SsoSettingsActions, SsoSettingsModel } from "./ssoSettingsTypes";

type SsoForm = {
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
};

export function useSsoSettingsState(model: SsoSettingsModel, actions: SsoSettingsActions) {
  const connection = model.connection;
  const [form, setForm] = useState(() => initialForm(connection));
  const [confirm, setConfirm] = useState<ConfirmRequest | null>(null);
  const [requiredConfirm, setRequiredConfirm] = useState(false);
  const access = accessState(model);
  const set = formSetters(setForm);
  const ask = (title: string, body: string, run: () => void) => setConfirm({ title, body, run });
  const requestRequired = () => setRequiredConfirm(true);
  const save = (nextEnforcement = form.enforcement) => {
    actions.onSave(connectionInput(form, nextEnforcement));
  };
  const draft: SsoDraft = {
    canChange: model.licensed,
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
    sessionHours: String(Math.round(seconds / 3600)),
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

function connectionInput(form: SsoForm, enforcement: string): SsoPutConnectionRequest {
  return {
    preset: form.preset as SsoPutConnectionRequest["preset"],
    displayName: form.displayName,
    issuer: form.issuer,
    clientId: form.clientId,
    clientSecret: form.clientSecret,
    jitEnabled: form.jitEnabled,
    defaultRole: form.defaultRole,
    enforcement: enforcement as SsoPutConnectionRequest["enforcement"],
    allowOwnerBypass: form.ownerBypass,
    maxSessionAgeSeconds: Math.round(Number(form.sessionHours) * 3600),
    idpLogoutEnabled: form.idpLogout,
  };
}
