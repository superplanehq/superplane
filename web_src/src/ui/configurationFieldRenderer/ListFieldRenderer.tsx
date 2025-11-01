import React from 'react'
import { Plus, Trash2 } from 'lucide-react'
import { Button } from '../button'
import { Input } from '../input'
import { FieldRendererProps } from './types'
import { ConfigurationFieldRenderer } from './index'

interface ExtendedFieldRendererProps extends FieldRendererProps {
  validationErrors?: Set<string>
  fieldPath?: string
}

export const ListFieldRenderer: React.FC<ExtendedFieldRendererProps> = ({
  field,
  value,
  onChange,
  domainId,
  domainType,
  hasError: _hasError,
  validationErrors,
  fieldPath = field.name || ''
}) => {
  const items = Array.isArray(value) ? value : []
  const listOptions = field.typeOptions?.list
  const itemDefinition = listOptions?.itemDefinition

  const addItem = () => {
    const newItem = itemDefinition?.type === 'object'
      ? {}
      : itemDefinition?.type === 'number'
      ? 0
      : ''
    onChange([...items, newItem])
  }

  const removeItem = (index: number) => {
    const newItems = items.filter((_, i) => i !== index)
    onChange(newItems.length > 0 ? newItems : undefined)
  }

  const updateItem = (index: number, newValue: unknown) => {
    const newItems = [...items]
    newItems[index] = newValue
    onChange(newItems)
  }

  return (
    <div className="space-y-3">
      {items.map((item, index) => (
        <div key={index} className="flex gap-2 items-center">
          <div className="flex-1">
            {itemDefinition?.type === 'object' && itemDefinition.schema ? (
              <div className="border border-gray-300 dark:border-zinc-700 rounded-md p-4 space-y-4">
                {itemDefinition.schema.map((schemaField) => {
                  const nestedFieldPath = `${fieldPath}[${index}].${schemaField.name}`
                  const hasNestedError = validationErrors?.has(nestedFieldPath) || false

                  return (
                    <ConfigurationFieldRenderer
                      key={schemaField.name}
                      field={schemaField}
                      value={item[schemaField.name!]}
                      onChange={(val) => {
                        const newItem = { ...item, [schemaField.name!]: val }
                        updateItem(index, newItem)
                      }}
                      allValues={item}
                      domainId={domainId}
                      domainType={domainType}
                      hasError={hasNestedError}
                    />
                  )
                })}
              </div>
            ) : (
              <Input
                type={itemDefinition?.type === 'number' ? 'number' : 'text'}
                value={item ?? ''}
                onChange={(e) => {
                  const val = itemDefinition?.type === 'number'
                    ? (e.target.value === '' ? undefined : Number(e.target.value))
                    : e.target.value
                  updateItem(index, val)
                }}
              />
            )}
          </div>
          <Button
            variant="ghost"
            size="icon"
            onClick={() => removeItem(index)}
            className="mt-1"
          >
            <Trash2 className="h-4 w-4 text-red-500" />
          </Button>
        </div>
      ))}
      <Button
        variant="outline"
        onClick={addItem}
        className="w-full mt-3"
      >
        <Plus className="h-4 w-4 mr-2" />
        Add Item
      </Button>
    </div>
  )
}
