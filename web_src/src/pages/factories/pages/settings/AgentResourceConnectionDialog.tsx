import { useEffect, useState } from "react";

import type {
  FactoriesFactoryAgentResource,
  FactoriesFactoryAgentResourceHeader,
  FactoryAgentResourceAuth,
} from "@/api-client";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { LoadingButton } from "@/components/ui/loading-button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { SecretKeyFieldRenderer } from "@/ui/configurationFieldRenderer/SecretKeyFieldRenderer";

import { AGENT_RESOURCES_COPY } from "./agentResourceCopy";

const NAME_PATTERN = /^[a-z][a-z0-9-]{0,62}$/;
const RESERVED_NAME = "superplane";
const SECRET_FIELD = {
  name: "header-secret",
  label: AGENT_RESOURCES_COPY.headerSecretLabel,
  placeholder: "Select credential",
};

type HeaderDraft = {
  name: string;
  secretName: string;
  secretKey: string;
};

export type AgentResourceConnectionDraft = {
  name: string;
  url: string;
  auth: FactoryAgentResourceAuth;
  headers: FactoriesFactoryAgentResourceHeader[];
};

function emptyHeader(): HeaderDraft {
  return { name: "", secretName: "", secretKey: "" };
}

function headersFromResource(resource?: FactoriesFactoryAgentResource): HeaderDraft[] {
  const headers = resource?.headers ?? [];
  if (headers.length === 0) {
    return [emptyHeader()];
  }
  return headers.map((header) => ({
    name: header.name ?? "",
    secretName: header.secretName ?? "",
    secretKey: header.secretKey ?? "",
  }));
}

function validateName(name: string): string {
  if (!name) {
    return AGENT_RESOURCES_COPY.nameRequired;
  }
  if (name === RESERVED_NAME) {
    return AGENT_RESOURCES_COPY.nameReserved;
  }
  if (!NAME_PATTERN.test(name)) {
    return AGENT_RESOURCES_COPY.nameInvalid;
  }
  return "";
}

function validateUrl(url: string): string {
  if (!url) {
    return AGENT_RESOURCES_COPY.urlRequired;
  }
  try {
    const parsed = new URL(url);
    if (parsed.protocol !== "https:") {
      return AGENT_RESOURCES_COPY.urlInvalid;
    }
  } catch {
    return AGENT_RESOURCES_COPY.urlInvalid;
  }
  return "";
}

function headerErrorForDraft(auth: FactoryAgentResourceAuth, headers: HeaderDraft[]): string {
  if (auth !== "AUTH_HEADERS") {
    return "";
  }
  const namedHeaders = headers.filter((header) => header.name.trim());
  const completeHeaders = namedHeaders.filter((header) => header.secretName && header.secretKey);
  if (completeHeaders.length !== namedHeaders.length) {
    return AGENT_RESOURCES_COPY.headerIncomplete;
  }
  return "";
}

export function AgentResourceConnectionDialog({
  open,
  organizationId,
  resource,
  defaults,
  isSaving,
  onClose,
  onSave,
}: {
  open: boolean;
  organizationId: string;
  resource?: FactoriesFactoryAgentResource;
  defaults?: Partial<AgentResourceConnectionDraft>;
  isSaving: boolean;
  onClose: () => void;
  onSave: (draft: AgentResourceConnectionDraft) => Promise<void>;
}) {
  const isEdit = Boolean(resource);
  const [name, setName] = useState("");
  const [url, setUrl] = useState("");
  const [auth, setAuth] = useState<FactoryAgentResourceAuth>("AUTH_OAUTH");
  const [headers, setHeaders] = useState<HeaderDraft[]>([emptyHeader()]);
  const [nameError, setNameError] = useState("");
  const [urlError, setUrlError] = useState("");
  const [headerError, setHeaderError] = useState("");

  useEffect(() => {
    if (!open) {
      return;
    }
    setName(resource?.name ?? defaults?.name ?? "");
    setUrl(resource?.url ?? defaults?.url ?? "");
    setAuth(
      resource ? (resource.auth === "AUTH_OAUTH" ? "AUTH_OAUTH" : "AUTH_HEADERS") : (defaults?.auth ?? "AUTH_OAUTH"),
    );
    setHeaders(headersFromResource(resource));
    setNameError("");
    setUrlError("");
    setHeaderError("");
  }, [open, resource, defaults]);

  const handleSave = async () => {
    const trimmedName = name.trim().toLowerCase();
    const trimmedUrl = url.trim();
    const nextNameError = validateName(trimmedName);
    const nextUrlError = validateUrl(trimmedUrl);
    const nextHeaderError = headerErrorForDraft(auth, headers);
    setNameError(nextNameError);
    setUrlError(nextUrlError);
    setHeaderError(nextHeaderError);
    if (nextNameError || nextUrlError || nextHeaderError) {
      return;
    }
    await onSave({
      name: trimmedName,
      url: trimmedUrl,
      auth,
      headers:
        auth === "AUTH_HEADERS"
          ? headers
              .filter((header) => header.name.trim() && header.secretName && header.secretKey)
              .map((header) => ({
                name: header.name.trim(),
                secretName: header.secretName,
                secretKey: header.secretKey,
              }))
          : [],
    });
  };

  return (
    <Dialog open={open} onOpenChange={(nextOpen) => !nextOpen && !isSaving && onClose()}>
      <DialogContent data-testid="agent-resource-connection-dialog">
        <DialogHeader>
          <DialogTitle>{isEdit ? AGENT_RESOURCES_COPY.editConnection : AGENT_RESOURCES_COPY.addConnection}</DialogTitle>
          <DialogDescription>{AGENT_RESOURCES_COPY.dialogDescription}</DialogDescription>
        </DialogHeader>
        <ConnectionDialogFields
          organizationId={organizationId}
          name={name}
          url={url}
          auth={auth}
          headers={headers}
          nameError={nameError}
          urlError={urlError}
          headerError={headerError}
          onNameChange={setName}
          onUrlChange={setUrl}
          onAuthChange={(value) => {
            const nextAuth: FactoryAgentResourceAuth = value === "AUTH_OAUTH" ? "AUTH_OAUTH" : "AUTH_HEADERS";
            setAuth(nextAuth);
            if (nextAuth === "AUTH_HEADERS" && headers.length === 0) {
              setHeaders([emptyHeader()]);
            }
          }}
          onHeadersChange={setHeaders}
        />
        <DialogFooter>
          <Button type="button" variant="outline" onClick={onClose} disabled={isSaving}>
            {AGENT_RESOURCES_COPY.cancel}
          </Button>
          <LoadingButton
            type="button"
            onClick={() => void handleSave()}
            loading={isSaving}
            data-testid="agent-resource-save"
          >
            {isEdit ? AGENT_RESOURCES_COPY.saveConnection : AGENT_RESOURCES_COPY.addConnection}
          </LoadingButton>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function ConnectionDialogFields({
  organizationId,
  name,
  url,
  auth,
  headers,
  nameError,
  urlError,
  headerError,
  onNameChange,
  onUrlChange,
  onAuthChange,
  onHeadersChange,
}: {
  organizationId: string;
  name: string;
  url: string;
  auth: FactoryAgentResourceAuth;
  headers: HeaderDraft[];
  nameError: string;
  urlError: string;
  headerError: string;
  onNameChange: (value: string) => void;
  onUrlChange: (value: string) => void;
  onAuthChange: (value: string) => void;
  onHeadersChange: (headers: HeaderDraft[]) => void;
}) {
  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="agent-resource-name">{AGENT_RESOURCES_COPY.nameLabel}</Label>
        <Input
          id="agent-resource-name"
          data-testid="agent-resource-name"
          value={name}
          onChange={(event) => onNameChange(event.target.value)}
          placeholder="docs"
          autoComplete="off"
        />
        <p className="text-[12px] text-muted-foreground">{AGENT_RESOURCES_COPY.nameHelper}</p>
        {nameError ? <p className="text-[12px] text-destructive">{nameError}</p> : null}
      </div>
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="agent-resource-url">{AGENT_RESOURCES_COPY.urlLabel}</Label>
        <Input
          id="agent-resource-url"
          data-testid="agent-resource-url"
          value={url}
          onChange={(event) => onUrlChange(event.target.value)}
          placeholder="https://mcp.example.com/mcp"
          autoComplete="off"
        />
        <p className="text-[12px] text-muted-foreground">{AGENT_RESOURCES_COPY.urlHelper}</p>
        {urlError ? <p className="text-[12px] text-destructive">{urlError}</p> : null}
      </div>
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="agent-resource-auth">{AGENT_RESOURCES_COPY.authLabel}</Label>
        <Select value={auth} onValueChange={onAuthChange}>
          <SelectTrigger id="agent-resource-auth" data-testid="agent-resource-auth" className="w-full">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="AUTH_OAUTH">{AGENT_RESOURCES_COPY.authSignIn}</SelectItem>
            <SelectItem value="AUTH_HEADERS">{AGENT_RESOURCES_COPY.authHeader}</SelectItem>
          </SelectContent>
        </Select>
      </div>
      {auth === "AUTH_HEADERS" ? (
        <ConnectionHeaderFields
          organizationId={organizationId}
          headers={headers}
          headerError={headerError}
          onHeadersChange={onHeadersChange}
        />
      ) : null}
    </div>
  );
}

function ConnectionHeaderFields({
  organizationId,
  headers,
  headerError,
  onHeadersChange,
}: {
  organizationId: string;
  headers: HeaderDraft[];
  headerError: string;
  onHeadersChange: (headers: HeaderDraft[]) => void;
}) {
  return (
    <div className="flex flex-col gap-3" data-testid="agent-resource-headers">
      <p className="text-[13px] font-medium">{AGENT_RESOURCES_COPY.headersLabel}</p>
      <p className="text-[12px] text-muted-foreground">{AGENT_RESOURCES_COPY.headersHelper}</p>
      {headers.map((header, index) => (
        <div key={index} className="grid gap-2 rounded-md border border-border p-3">
          <div className="flex flex-col gap-1.5">
            <Label htmlFor={`agent-resource-header-name-${index}`}>{AGENT_RESOURCES_COPY.headerNameLabel}</Label>
            <Input
              id={`agent-resource-header-name-${index}`}
              data-testid={`agent-resource-header-name-${index}`}
              value={header.name}
              onChange={(event) =>
                onHeadersChange(
                  headers.map((entry, entryIndex) =>
                    entryIndex === index ? { ...entry, name: event.target.value } : entry,
                  ),
                )
              }
              placeholder="Authorization"
            />
          </div>
          <SecretKeyFieldRenderer
            field={{ ...SECRET_FIELD, name: `header-secret-${index}` }}
            isRequired={false}
            value={
              header.secretName && header.secretKey ? { secret: header.secretName, key: header.secretKey } : undefined
            }
            onChange={(value) =>
              onHeadersChange(
                headers.map((entry, entryIndex) =>
                  entryIndex === index
                    ? { ...entry, secretName: value?.secret ?? "", secretKey: value?.key ?? "" }
                    : entry,
                ),
              )
            }
            organizationId={organizationId}
          />
          {headers.length > 1 ? (
            <Button
              type="button"
              variant="ghost"
              size="sm"
              className="justify-start px-0"
              onClick={() => onHeadersChange(headers.filter((_, entryIndex) => entryIndex !== index))}
            >
              {AGENT_RESOURCES_COPY.removeHeader}
            </Button>
          ) : null}
        </div>
      ))}
      <Button type="button" variant="outline" size="sm" onClick={() => onHeadersChange([...headers, emptyHeader()])}>
        {AGENT_RESOURCES_COPY.addHeader}
      </Button>
      {headerError ? <p className="text-[12px] text-destructive">{headerError}</p> : null}
    </div>
  );
}
