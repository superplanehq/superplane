import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { ExternalLink, Loader2 } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { useQueryClient } from "@tanstack/react-query";
import { IntegrationResourceFieldRenderer } from "@/ui/configurationFieldRenderer/IntegrationResourceFieldRenderer";
import { SecretPickerFieldRenderer } from "@/ui/configurationFieldRenderer/SecretPickerFieldRenderer";
import type { AppEntry } from "./AppDetailModal";
import { IntegrationIcons } from "./AppDetailModal";
import { IntegrationsSection, type IntegrationSelections } from "./InstallIntegrationsSection";
import { useInstallPreviewData } from "./useInstallPreviewData";
import { useInstallAction } from "./useInstallAction";
import type { InstallParam } from "../install/types";

function allIntegrationsSelected(integrations: string[], selections: IntegrationSelections): boolean {
  return integrations.length === 0 || integrations.every((name) => selections[name]);
}

function checkRequiredParams(params: InstallParam[], values: Record<string, string>): boolean {
  return params.filter((p) => p.required && !p.default).every((p) => (values[p.name] ?? "").trim() !== "");
}

function normalizeResourceValue(val: string | string[] | undefined): string {
  if (typeof val === "string") return val;
  if (Array.isArray(val)) return val[0] ?? "";
  return "";
}

interface InstallProgressPanelProps {
  app: AppEntry;
  organizationId?: string;
  canvasName?: string;
  /** When provided, the panel renders the canvas name as an editable input. */
  onCanvasNameChange?: (name: string) => void;
  /** When true, skips preview fetch (caller already loaded data via preloaded props). */
  skipPreviewFetch?: boolean;
  preloadedIntegrations?: string[];
  preloadedParams?: InstallParam[];
  onClose: () => void;
}

export function InstallProgressPanel({
  app,
  organizationId: propOrgId,
  canvasName,
  onCanvasNameChange,
  skipPreviewFetch,
  preloadedIntegrations,
  preloadedParams,
  onClose,
}: InstallProgressPanelProps) {
  const { organizationId: routeOrgId } = useParams<{ organizationId: string }>();
  const organizationId = propOrgId || routeOrgId;
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const preview = useInstallPreviewData({
    repo: app.repo,
    skipFetch: skipPreviewFetch,
    preloadedIntegrations,
    preloadedParams,
  });
  const [integrationSelections, setIntegrationSelections] = useState<IntegrationSelections>({});

  const resetParamValuesRef = useRef(preview.resetParamValues);
  resetParamValuesRef.current = preview.resetParamValues;
  useEffect(() => {
    setIntegrationSelections({});
    resetParamValuesRef.current();
  }, [organizationId]);

  const { doInstall, isInstalling } = useInstallAction({
    organizationId,
    app,
    canvasName,
    installParams: preview.installParams,
    paramValues: preview.paramValues,
    integrationSelections,
    queryClient,
    navigate,
  });

  const integrations = preview.detectedIntegrations.length > 0 ? preview.detectedIntegrations : app.integrations;
  const hasIntegrations = integrations.length > 0;
  const hasParams = preview.installParams.length > 0;
  const readyToInstall = !!organizationId && !isInstalling && !preview.previewLoading && !preview.previewError;
  const canInstall =
    readyToInstall &&
    allIntegrationsSelected(integrations, integrationSelections) &&
    checkRequiredParams(preview.installParams, preview.paramValues);
  const canSkip = readyToInstall;

  return (
    <div className="mt-4 rounded-lg bg-white p-5 outline outline-slate-950/10 animate-in slide-in-from-top-2 dark:bg-gray-900">
      <AppInfoHeader app={app} integrations={integrations} />

      {onCanvasNameChange && (
        <div className="mb-5 border-t border-slate-100 pt-4">
          <AppNameSection value={canvasName ?? ""} onChange={onCanvasNameChange} />
        </div>
      )}

      {hasIntegrations && (
        <div className="mb-5 border-t border-slate-100 pt-4">
          <IntegrationsSection
            integrations={integrations}
            organizationId={organizationId ?? ""}
            selections={integrationSelections}
            onSelectionsChange={setIntegrationSelections}
          />
        </div>
      )}

      <PreviewStatus loading={preview.previewLoading} error={preview.previewError} />

      {!preview.previewLoading && hasParams && (
        <div className="mb-5 border-t border-slate-100 pt-4">
          <ParamsSection
            params={preview.installParams}
            values={preview.paramValues}
            onChange={preview.setParamValues}
            organizationId={organizationId}
            integrationSelections={integrationSelections}
          />
        </div>
      )}

      {!preview.previewLoading && (
        <InstallActions
          canInstall={canInstall}
          canSkip={canSkip}
          isInstalling={isInstalling}
          onInstall={() => void doInstall(false)}
          onSkip={() => void doInstall(true)}
          onClose={onClose}
        />
      )}
    </div>
  );
}

// ─── Sub-components ──────────────────────────────────────────────────────────

function PreviewStatus({ loading, error }: { loading: boolean; error: string | null }) {
  if (loading) {
    return (
      <div className="mb-5 border-t border-slate-100 pt-4">
        <div className="flex items-center gap-2 text-xs text-slate-400">
          <Loader2 className="h-3.5 w-3.5 animate-spin" />
          Loading configuration...
        </div>
      </div>
    );
  }
  if (error) {
    return (
      <div className="mb-5 border-t border-slate-100 pt-4">
        <p className="text-xs text-red-600">{error}</p>
      </div>
    );
  }
  return null;
}

function InstallButton({
  isInstalling,
  disabled,
  onClick,
}: {
  isInstalling: boolean;
  disabled: boolean;
  onClick: () => void;
}) {
  return (
    <Button size="sm" onClick={onClick} disabled={disabled}>
      {isInstalling ? (
        <>
          <Loader2 className="h-3.5 w-3.5 animate-spin mr-1" />
          Installing...
        </>
      ) : (
        "Install"
      )}
    </Button>
  );
}

function InstallActions({
  canInstall,
  canSkip,
  isInstalling,
  onInstall,
  onSkip,
  onClose,
}: {
  canInstall: boolean;
  canSkip: boolean;
  isInstalling: boolean;
  onInstall: () => void;
  onSkip: () => void;
  onClose: () => void;
}) {
  return (
    <div className="flex items-center gap-2 border-t border-slate-100 pt-4">
      <InstallButton isInstalling={isInstalling} disabled={!canInstall} onClick={onInstall} />
      <Button variant="outline" size="sm" onClick={onSkip} disabled={!canSkip}>
        Just take me there
      </Button>
      <Button variant="ghost" size="sm" onClick={onClose} disabled={isInstalling}>
        Cancel
      </Button>
    </div>
  );
}

function AppInfoHeader({ app, integrations }: { app: AppEntry; integrations: string[] }) {
  return (
    <div className="mb-5">
      <div className="flex items-start gap-3">
        <div className="min-w-0 flex-1">
          <h3 className="text-sm font-semibold text-slate-900">{app.title}</h3>
          <div className="mt-1 flex flex-wrap items-center gap-2">
            <IntegrationIcons integrations={integrations} />
            {app.tags.map((tag) => (
              <span key={tag} className="rounded-full bg-slate-100 px-2 py-0.5 text-[10px] font-medium text-slate-500">
                {tag}
              </span>
            ))}
          </div>
        </div>
        <a
          href={`https://${app.repo}`}
          target="_blank"
          rel="noopener noreferrer"
          className="flex items-center gap-1 text-[10px] font-medium text-slate-400 hover:text-slate-600 shrink-0"
        >
          <ExternalLink className="h-3 w-3" />
          GitHub
        </a>
      </div>
      {app.description && <p className="mt-2 text-xs leading-relaxed text-slate-600">{app.description}</p>}
      {app.requirements.length > 0 && <RequirementsList requirements={app.requirements} />}
    </div>
  );
}

function AppNameSection({ value, onChange }: { value: string; onChange: (name: string) => void }) {
  return (
    <div className="space-y-1">
      <Label htmlFor="install-app-name" className="text-xs">
        App name
        <span className="text-red-500 ml-0.5">*</span>
      </Label>
      <Input
        id="install-app-name"
        value={value}
        onChange={(event) => onChange(event.target.value)}
        placeholder="Give this app a name"
        className="h-8 text-xs"
      />
    </div>
  );
}

function RequirementsList({ requirements }: { requirements: string[] }) {
  return (
    <ul className="mt-2 space-y-0.5">
      {requirements.map((req) => (
        <li key={req} className="flex items-start gap-1.5 text-xs text-slate-500">
          <span className="mt-1.5 h-1 w-1 shrink-0 rounded-full bg-slate-300" />
          {req}
        </li>
      ))}
    </ul>
  );
}

function ParamsSection({
  params,
  values,
  onChange,
  organizationId,
  integrationSelections,
}: {
  params: InstallParam[];
  values: Record<string, string>;
  onChange: React.Dispatch<React.SetStateAction<Record<string, string>>>;
  organizationId?: string;
  integrationSelections: IntegrationSelections;
}) {
  return (
    <>
      <p className="text-xs font-semibold text-slate-700 mb-3">Configuration</p>
      <div className="space-y-3">
        {params.map((param) => (
          <div key={param.name} className="space-y-1">
            <Label htmlFor={`param-${param.name}`} className="text-xs">
              {param.label}
              {param.required && <span className="text-red-500 ml-0.5">*</span>}
            </Label>
            {param.type === "integration-resource" && param.integration && param.resourceType ? (
              <IntegrationResourceFieldRenderer
                field={{
                  name: param.name,
                  label: param.label,
                  type: "integration-resource",
                  placeholder: param.placeholder,
                  required: param.required,
                  typeOptions: { resource: { type: param.resourceType, useNameAsValue: param.useNameAsValue } },
                }}
                value={values[param.name]}
                onChange={(val) => onChange((prev) => ({ ...prev, [param.name]: normalizeResourceValue(val) }))}
                organizationId={organizationId}
                integrationId={integrationSelections[param.integration]?.id}
              />
            ) : param.type === "secret_picker" ? (
              <SecretPickerFieldRenderer
                id={`param-${param.name}`}
                placeholder={param.placeholder}
                required={param.required}
                value={values[param.name]}
                onChange={(val) => onChange((prev) => ({ ...prev, [param.name]: val }))}
                organizationId={organizationId}
              />
            ) : (
              <Input
                id={`param-${param.name}`}
                value={values[param.name] ?? ""}
                placeholder={param.placeholder}
                className="h-8 text-xs"
                onChange={(e) => onChange((prev) => ({ ...prev, [param.name]: e.target.value }))}
              />
            )}
            {param.description && <p className="text-[10px] text-slate-400">{param.description}</p>}
          </div>
        ))}
      </div>
    </>
  );
}
