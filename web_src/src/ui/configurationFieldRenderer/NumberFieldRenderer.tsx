import React from 'react'
import { Input } from '../input'
import { FieldRendererProps } from './types'

export const NumberFieldRenderer: React.FC<FieldRendererProps> = ({ field, value, onChange, hasError }) => {
  const numberOptions = field.typeOptions?.number

  return (
    <Input
      type="number"
      value={(value as string | number) ?? (field.defaultValue as string) ?? ''}
      onChange={(e) => {
        const val = e.target.value === '' ? undefined : Number(e.target.value)
        onChange(val)
      }}
      placeholder={`Enter ${field.name}`}
      min={numberOptions?.min}
      max={numberOptions?.max}
      className={hasError ? 'border-red-500 border-2' : ''}
    />
  )
}
