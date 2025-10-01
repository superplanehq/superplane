import { Input } from '../Input/input';
import { DynamicFormField } from './DynamicFormField';
import type { ArrayFieldProps } from './types';

export function ArrayField({
  field,
  value,
  onChange,
  disabled,
  parentPath,
  context,
}: ArrayFieldProps) {
  const items = value || [];

  const addItem = () => {
    if (field.itemType === 'object' && field.fields) {
      // Initialize with empty object
      onChange([...items, {}]);
    } else if (field.itemType === 'number') {
      onChange([...items, 0]);
    } else {
      onChange([...items, '']);
    }
  };

  const removeItem = (index: number) => {
    onChange(items.filter((_, i) => i !== index));
  };

  const updateItem = (index: number, newValue: any) => {
    const newItems = [...items];
    newItems[index] = newValue;
    onChange(newItems);
  };

  return (
    <div className="space-y-2">
      {items.map((item, index) => (
        <div key={index} className="flex gap-2 items-start">
          <div className="flex-1">
            {field.itemType === 'object' && field.fields ? (
              <div className="p-3 border border-zinc-200 rounded-lg space-y-3">
                {field.fields.map((nestedField) => (
                  <DynamicFormField
                    key={nestedField.name}
                    field={nestedField}
                    value={item?.[nestedField.name || '']}
                    onChange={(newValue: any) => {
                      updateItem(index, {
                        ...item,
                        [nestedField.name || '']: newValue,
                      });
                    }}
                    parentPath={`${parentPath}[${index}]`}
                    disabled={disabled}
                    context={context}
                  />
                ))}
              </div>
            ) : (
              <Input
                type={field.itemType === 'number' ? 'number' : 'text'}
                value={item || ''}
                onChange={(e) => {
                  const newValue =
                    field.itemType === 'number'
                      ? e.target.value
                        ? Number(e.target.value)
                        : undefined
                      : e.target.value;
                  updateItem(index, newValue);
                }}
                disabled={disabled}
              />
            )}
          </div>
          <button
            type="button"
            onClick={() => removeItem(index)}
            disabled={disabled}
            className="px-3 py-2 text-sm text-red-600 hover:text-red-700"
          >
            Remove
          </button>
        </div>
      ))}
      <button
        type="button"
        onClick={addItem}
        disabled={disabled}
        className="text-sm text-blue-600 hover:text-blue-700"
      >
        + Add Item
      </button>
    </div>
  );
}
