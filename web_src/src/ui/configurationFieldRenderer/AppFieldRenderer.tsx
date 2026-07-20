import { useMemo } from "react";
import { useParams } from "react-router-dom";
import { AutoCompleteSelect, type AutoCompleteOption } from "@/components/AutoCompleteSelect";
import { Select, SelectTrigger, SelectValue } from "@/components/ui/select";
import type { ConfigurationField } from "@/api-client";
import { useCanvases } from "@/hooks/useCanvasData";
import { toTestId } from "@/lib/testID";

interface AppFieldRendererProps {
  field: ConfigurationField;
  value: string | undefined;
  onChange: (value: string | undefined) => void;
  organizationId: string;
  readOnly?: boolean;
}

export function AppFieldRenderer({ field, value, onChange, organizationId, readOnly = false }: AppFieldRendererProps) {
  const { appId: currentAppId } = useParams<{ appId?: string }>();
  const { data: canvases, isLoading, error } = useCanvases(organizationId);
  const allowSelf = field.typeOptions?.app?.allowSelf ?? false;

  const options: AutoCompleteOption[] = useMemo(() => {
    if (!canvases?.length) {
      return [];
    }

    return canvases
      .filter((canvas) => {
        if (!canvas.id || !canvas.name) {
          return false;
        }

        if (!allowSelf && canvas.id === currentAppId) {
          return false;
        }

        return true;
      })
      .map((canvas) => ({
        value: canvas.id!,
        label: canvas.name!,
      }))
      .sort((left, right) => left.label.localeCompare(right.label));
  }, [allowSelf, canvases, currentAppId]);

  const selectedValue = useMemo(() => {
    if (!value) {
      return "";
    }

    const matchedCanvas = canvases?.find((canvas) => canvas.id === value || canvas.name === value);
    return matchedCanvas?.id ?? value;
  }, [canvases, value]);

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
          {allowSelf
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
