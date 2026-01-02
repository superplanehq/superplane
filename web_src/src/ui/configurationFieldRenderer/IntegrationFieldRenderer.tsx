import React from "react";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../select";
import { useIntegrations } from "@/hooks/useIntegrations";
import { AuthorizationDomainType, ConfigurationField } from "../../api-client";
import { toTestId } from "@/utils/testID";

interface IntegrationFieldRendererProps {
  field: ConfigurationField;
  value: string;
  onChange: (value: string | undefined) => void;
  domainId?: string;
  domainType?: AuthorizationDomainType;
}

export const IntegrationFieldRenderer = ({
  field,
  value,
  onChange,
  domainId,
  domainType,
}: IntegrationFieldRendererProps) => {
  const integrationType = field.typeOptions?.integration?.type;

  // Fetch integrations if we have the required context
  const { data: integrations, isLoading } = useIntegrations(domainId ?? "", domainType ?? "DOMAIN_TYPE_ORGANIZATION");

  // Filter integrations by type
  const filteredIntegrations = React.useMemo(() => {
    if (!integrations || !integrationType) return [];
    return integrations.filter((integration) => integration.spec?.type === integrationType);
  }, [integrations, integrationType]);

  if (!domainId || !domainType) {
    return (
      <div className="text-sm text-red-500 dark:text-red-400">
        Integration field requires domainId and domainType props
      </div>
    );
  }

  if (isLoading) {
    return <div className="text-sm text-gray-500 dark:text-gray-400">Loading {integrationType} integrations...</div>;
  }

  if (filteredIntegrations.length === 0) {
    return (
      <div className="space-y-2">
        <Select disabled>
          <SelectTrigger className="w-full">
            <SelectValue placeholder="No integrations available" />
          </SelectTrigger>
        </Select>
        <p className="text-xs text-gray-500 dark:text-gray-400">
          No {integrationType} integrations found. Please create one in the settings.
        </p>
      </div>
    );
  }

  return (
    <Select value={value ?? ""} onValueChange={(val) => onChange(val || undefined)}>
      <SelectTrigger className="w-full" data-testid={toTestId(`integration-field-${field.name}`)}>
        <SelectValue placeholder={`Select ${integrationType} integration`} />
      </SelectTrigger>
      <SelectContent>
        {filteredIntegrations.map((integration) => (
          <SelectItem key={integration.metadata?.id} value={integration.metadata?.id ?? ""}>
            {integration.metadata?.name} ({integration.spec?.type})
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
};
