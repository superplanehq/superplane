import { ComponentsConfigurationField } from '../../api-client'

export interface FieldRendererProps {
  field: ComponentsConfigurationField
  value: unknown
  onChange: (value: unknown) => void
  allValues?: Record<string, unknown>
  domainId?: string
  domainType?: "DOMAIN_TYPE_CANVAS" | "DOMAIN_TYPE_ORGANIZATION"
  hasError?: boolean
}
