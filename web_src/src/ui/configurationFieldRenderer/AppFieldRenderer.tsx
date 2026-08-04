import { useMemo } from "react";
import { useParams } from "react-router-dom";
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

export function AppFieldRenderer({ field, value, onChange, organizationId, readOnly = false }: AppFieldRendererProps) {
  const { appId: currentAppId } = useParams<{ appId?: string }>();
  const { data: currentCanvas } = useCanvas(organizationId, currentAppId ?? "", {
    enabled: Boolean(currentAppId),
    staleTime: Infinity,
  });
  const factoryId = currentCanvas?.metadata?.factoryId;
  const allowSelf = field.typeOptions?.app?.allowSelf ?? false;

  const orgCanvasesQuery = useCanvases(organizationId);
  const factoryAppsQuery = useFactoryApps(organizationId, factoryId ?? "");

  const isFactoryContext = Boolean(factoryId);
  const { data: apps, isLoading, error } = isFactoryContext ? factoryAppsQuery : orgCanvasesQuery;

  const options = useMemo(
    () => buildAppOptions(apps, allowSelf, currentAppId),
    [allowSelf, apps, currentAppId],
  );

  const selectedValue = useMemo(() => {
    if (!value) {
      return "";
    }

    const matchedApp = apps?.find((app) => app.id === value || app.name === value);
    return matchedApp?.id ?? value;
  }, [apps, value]);

  if (error) {
    return (
      <div className="text-sm text-red-500 dark:text-red-400">
        Failed to load apps: {error instanceof Error ? error.message : "Unknown error"}
      </div>
    );
  }

  if (isLoading) {
    return (
      <div data-testid={toTestId(`app-field-${field.name}`)}>
        <Select value="" disabled>
          <SelectTrigger className="w-full">
            <SelectValue placeholder="Loading apps..." />
          </SelectTrigger>
        </Select>
      </div>
    );
  }

  if (options.length === 0) {
    return (
      <div data-testid={toTestId(`app-field-${field.name}`)} className="space-y-2">
        <Select value="" disabled>
          <SelectTrigger className="w-full">
            <SelectValue placeholder="No apps available" />
          </SelectTrigger>
        </Select>
        <p className="text-xs text-gray-500 dark:text-gray-400">
          {isFactoryContext
            ? allowSelf
              ? "Select another app in this factory to invoke."
              : "Create another app in this factory to subscribe to its events."
            : allowSelf
              ? "Select an app in this organization to invoke."
              : "Create another app in this organization to subscribe to its events."}
        </p>
      </div>
    );
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
