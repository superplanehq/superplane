import { ComponentsConfigurationField } from '../../api-client'

export interface FieldRendererProps {
  field: ComponentsConfigurationField
  value: any
  onChange: (value: any) => void
  allValues?: Record<string, any>
}
