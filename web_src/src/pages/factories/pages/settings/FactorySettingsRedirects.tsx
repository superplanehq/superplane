import { useAccount } from "@/contexts/useAccount";
import { useFactories } from "@/hooks/useFactoryData";
import {
  factorySettingsSectionPath,
  factorySettingsWorkspaceGeneralPath,
  type FactorySettingsScope,
} from "../../lib/factoryPagePaths";
import { pickInitialFactory, readLastVisitedFactory } from "../../lib/lastVisitedFactory";
import { Navigate, useLocation, useParams } from "react-router";

type FactorySettingsDestination = {
  scope: FactorySettingsScope;
  section: string;
};

const WORKSPACE_GENERAL: FactorySettingsDestination = { scope: "workspace", section: "general" };
const ORGANIZATION_GENERAL: FactorySettingsDestination = { scope: "organization", section: "general" };

const LEGACY_WORKSPACE_SETTINGS: Record<string, FactorySettingsDestination> = {
  general: { scope: "workspace", section: "general" },
  automations: { scope: "workspace", section: "automations" },
  models: { scope: "workspace", section: "models" },
  usage: { scope: "workspace", section: "spending" },
  repositories: { scope: "workspace", section: "repository" },
  repository: { scope: "workspace", section: "repository" },
  profile: { scope: "account", section: "general" },
  notifications: { scope: "account", section: "notifications" },
  members: { scope: "organization", section: "members" },
  integrations: { scope: "organization", section: "integrations" },
  secrets: { scope: "organization", section: "secrets" },
};

const LEGACY_ORGANIZATION_SETTINGS: Record<string, FactorySettingsDestination> = {
  general: { scope: "organization", section: "general" },
  members: { scope: "organization", section: "members" },
  "api-keys": { scope: "organization", section: "api-keys" },
  integrations: { scope: "organization", section: "integrations" },
  "llm-spend": { scope: "organization", section: "spending" },
  secrets: { scope: "organization", section: "secrets" },
};

function splitLegacyPath(rest: string | undefined) {
  return rest?.split("/").filter(Boolean) ?? [];
}

function factorySettingsDestination(
  organizationId: string,
  factoryKey: string,
  mapping: Record<string, FactorySettingsDestination>,
  rest: string | undefined,
  fallback: FactorySettingsDestination,
) {
  const [section, ...detailSegments] = splitLegacyPath(rest);
  const mappedDestination = section ? mapping[section] : undefined;
  const destination = mappedDestination ?? fallback;
  const basePath = factorySettingsSectionPath(organizationId, factoryKey, destination.scope, destination.section);
  if (!mappedDestination || detailSegments.length === 0) {
    return basePath;
  }
  return `${basePath}/${detailSegments.join("/")}`;
}

export function LegacyFactorySettingsRedirect() {
  const {
    organizationId,
    factoryKey,
    "*": rest,
  } = useParams<{
    organizationId: string;
    factoryKey: string;
    "*": string;
  }>();
  const location = useLocation();

  if (!organizationId || !factoryKey) {
    return <Navigate to="/" replace />;
  }

  const destination = factorySettingsDestination(
    organizationId,
    factoryKey,
    LEGACY_WORKSPACE_SETTINGS,
    rest,
    WORKSPACE_GENERAL,
  );
  return <Navigate to={`${destination}${location.search}`} replace />;
}

export function LegacyFactoryOrganizationSettingsRedirect() {
  const {
    organizationId,
    factoryKey,
    "*": rest,
  } = useParams<{
    organizationId: string;
    factoryKey: string;
    "*": string;
  }>();
  const location = useLocation();

  if (!organizationId || !factoryKey) {
    return <Navigate to="/" replace />;
  }

  const destination = factorySettingsDestination(
    organizationId,
    factoryKey,
    LEGACY_ORGANIZATION_SETTINGS,
    rest,
    ORGANIZATION_GENERAL,
  );
  return <Navigate to={`${destination}${location.search}`} replace />;
}

export function LegacyOrganizationSettingsRedirect({ destination = "general" }: { destination?: string }) {
  const { organizationId, "*": rest } = useParams<{ organizationId: string; "*": string }>();
  const { account } = useAccount();
  const { data: factories = [], isLoading } = useFactories(organizationId ?? "");
  const location = useLocation();

  if (!organizationId) {
    return <Navigate to="/" replace />;
  }

  if (isLoading) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-background text-foreground">
        <p className="text-[13px] text-muted-foreground">Loading settings…</p>
      </div>
    );
  }

  const lastVisitedId = account?.id ? readLastVisitedFactory(account.id, organizationId) : null;
  const factory = pickInitialFactory(factories, lastVisitedId);
  if (!factory?.key) {
    return <Navigate to={`/${organizationId}/workspaces`} replace />;
  }

  const targetRest = rest || destination;
  const path = factorySettingsDestination(
    organizationId,
    factory.key,
    LEGACY_ORGANIZATION_SETTINGS,
    targetRest,
    ORGANIZATION_GENERAL,
  );
  return <Navigate to={`${path}${location.search}`} replace />;
}

export function LegacyFactorySettingsIndexRedirect() {
  const { organizationId, factoryKey } = useParams<{ organizationId: string; factoryKey: string }>();
  const location = useLocation();
  if (!organizationId || !factoryKey) {
    return <Navigate to="/" replace />;
  }
  return (
    <Navigate to={`${factorySettingsWorkspaceGeneralPath(organizationId, factoryKey)}${location.search}`} replace />
  );
}
