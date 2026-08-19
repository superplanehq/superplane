import { useMemo } from "react";
import { useParams } from "react-router";
import { AutoCompleteSelect, type AutoCompleteOption } from "@/components/AutoCompleteSelect";
import { Select, SelectTrigger, SelectValue } from "@/components/ui/select";
import type { ConfigurationField } from "@/api-client";
import { useCanvas, useCanvases } from "@/hooks/useCanvasData";
import { useFactoryApps } from "@/hooks/useFactoryData";
import { toTestId } from "@/lib/testID";

interface AppFieldRendererProps {
  field: ConfigurationField;
  value: string | undefined;
  onChange: (value: string | undefined) => void;
  organizationId: string;
  readOnly?: boolean;
}

type AppOptionSource = {
  id?: string;
  name?: string;
};

type AppFieldSources = {
  apps: AppOptionSource[] | undefined;
  isLoading: boolean;
  error: unknown;
  isFactoryContext: boolean;
};

function buildAppOptions(
  apps: AppOptionSource[] | undefined,
  allowSelf: boolean,
  currentAppId: string | undefined,
): AutoCompleteOption[] {
  if (!apps?.length) {
    return [];
  }

  return apps
    .filter((app) => {
      if (!app.id || !app.name) {
        return false;
      }

      if (!allowSelf && app.id === currentAppId) {
        return false;
      }

      return true;
    })
    .map((app) => ({
      value: app.id!,
      label: app.name!,
    }))
    .sort((left, right) => left.label.localeCompare(right.label));
}

function resolveSelectedAppValue(apps: AppOptionSource[] | undefined, value: string | undefined): string {
  if (!value) {
    return "";
  }

  const matchedApp = apps?.find((app) => app.id === value || app.name === value);
  return matchedApp?.id ?? value;
}

function getEmptyStateMessage(isFactoryContext: boolean, allowSelf: boolean): string {
  if (isFactoryContext) {
    return allowSelf
      ? "Select another app in this factory to invoke."
      : "Create another app in this factory to subscribe to its events.";
  }

  return allowSelf
    ? "Select an app in this organization to invoke."
    : "Create another app in this organization to subscribe to its events.";
}

function useAppFieldSources(organizationId: string, currentAppId: string | undefined): AppFieldSources {
  const needsCanvasContext = Boolean(currentAppId);
  const canvasQuery = useCanvas(organizationId, currentAppId ?? "", {
    enabled: needsCanvasContext,
    staleTime: Infinity,
  });

  const factoryId = canvasQuery.data?.metadata?.factoryId;
  const isFactoryContext = Boolean(factoryId);

  const orgCanvasesQuery = useCanvases(organizationId);
  const factoryAppsQuery = useFactoryApps(organizationId, factoryId ?? "");

  if (needsCanvasContext && canvasQuery.isLoading) {
    return {
      apps: undefined,
      isLoading: true,
      error: null,
      isFactoryContext: false,
    };
  }

  if (needsCanvasContext && canvasQuery.error) {
    return {
      apps: undefined,
      isLoading: false,
      error: canvasQuery.error,
      isFactoryContext: false,
    };
  }

  const activeQuery = isFactoryContext ? factoryAppsQuery : orgCanvasesQuery;

  return {
    apps: activeQuery.data,
    isLoading: activeQuery.isLoading,
    error: activeQuery.error,
    isFactoryContext,
  };
}

function AppFieldLoadError({ error }: { error: unknown }) {
  const message = error instanceof Error ? error.message : "Unknown error";

  return <div className="text-sm text-red-500 dark:text-red-400">Failed to load apps: {message}</div>;
}

function AppFieldLoading({ fieldName }: { fieldName: string | undefined }) {
  return (
    <div data-testid={toTestId(`app-field-${fieldName}`)}>
      <Select value="" disabled>
        <SelectTrigger className="w-full">
          <SelectValue placeholder="Loading apps..." />
        </SelectTrigger>
      </Select>
    </div>
  );
}

function AppFieldEmpty({ fieldName, message }: { fieldName: string | undefined; message: string }) {
  return (
    <div data-testid={toTestId(`app-field-${fieldName}`)} className="space-y-2">
      <Select value="" disabled>
        <SelectTrigger className="w-full">
          <SelectValue placeholder="No apps available" />
        </SelectTrigger>
      </Select>
      <p className="text-xs text-gray-500 dark:text-gray-400">{message}</p>
    </div>
  );
}

export function AppFieldRenderer({ field, value, onChange, organizationId, readOnly = false }: AppFieldRendererProps) {
  const { appId: currentAppId } = useParams<{ appId?: string }>();
  const allowSelf = field.typeOptions?.app?.allowSelf ?? false;
  const { apps, isLoading, error, isFactoryContext } = useAppFieldSources(organizationId, currentAppId);

  const options = useMemo(() => buildAppOptions(apps, allowSelf, currentAppId), [allowSelf, apps, currentAppId]);

  const selectedValue = useMemo(() => resolveSelectedAppValue(apps, value), [apps, value]);
  const emptyStateMessage = useMemo(
    () => getEmptyStateMessage(isFactoryContext, allowSelf),
    [allowSelf, isFactoryContext],
  );

  if (error) {
    return <AppFieldLoadError error={error} />;
  }

  if (isLoading) {
    return <AppFieldLoading fieldName={field.name} />;
  }

  if (options.length === 0) {
    return <AppFieldEmpty fieldName={field.name} message={emptyStateMessage} />;
  }

  return (
    <div data-testid={toTestId(`app-field-${field.name}`)}>
      <AutoCompleteSelect
        options={options}
        value={selectedValue}
        onChange={(nextValue) => onChange(nextValue || undefined)}
        placeholder={field.placeholder ?? "Select app"}
        disabled={readOnly}
      />
    </div>
  );
}
