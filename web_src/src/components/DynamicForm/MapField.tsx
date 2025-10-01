import { Input } from '../Input/input';
import type { MapFieldProps } from './types';

export function MapField({
  value,
  onChange,
  placeholder,
  disabled,
}: MapFieldProps) {
  const entries = Object.entries(value || {});

  const addEntry = () => {
    onChange({ ...value, '': '' });
  };

  const removeEntry = (key: string) => {
    const newValue = { ...value };
    delete newValue[key];
    onChange(newValue);
  };

  const updateEntry = (oldKey: string, newKey: string, newValue: string) => {
    const entries = { ...value };
    if (oldKey !== newKey && oldKey in entries) {
      delete entries[oldKey];
    }
    entries[newKey] = newValue;
    onChange(entries);
  };

  return (
    <div className="space-y-2">
      {entries.map(([key, val], index) => (
        <div key={index} className="flex gap-2">
          <Input
            type="text"
            value={key}
            onChange={(e) => updateEntry(key, e.target.value, val)}
            placeholder="Key"
            disabled={disabled}
            className="flex-1"
          />
          <Input
            type="text"
            value={val}
            onChange={(e) => updateEntry(key, key, e.target.value)}
            placeholder="Value"
            disabled={disabled}
            className="flex-1"
          />
          <button
            type="button"
            onClick={() => removeEntry(key)}
            disabled={disabled}
            className="px-3 py-2 text-sm text-red-600 hover:text-red-700"
          >
            Remove
          </button>
        </div>
      ))}
      <button
        type="button"
        onClick={addEntry}
        disabled={disabled}
        className="text-sm text-blue-600 hover:text-blue-700"
      >
        + Add {placeholder || 'Entry'}
      </button>
    </div>
  );
}
