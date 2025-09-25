interface SwitchProps {
  checked: boolean;
  onChange: (checked: boolean) => void;
  color?: 'blue' | 'green' | 'indigo';
  disabled?: boolean;
  className?: string;
  'aria-label'?: string;
}

export function Switch({
  checked,
  onChange,
  color = 'blue',
  disabled = false,
  className = '',
  'aria-label': ariaLabel
}: SwitchProps) {
  const colorClasses = {
    blue: checked ? 'bg-blue-600' : 'bg-gray-200 dark:bg-gray-700',
    green: checked ? 'bg-green-600' : 'bg-gray-200 dark:bg-gray-700',
    indigo: checked ? 'bg-indigo-600' : 'bg-gray-200 dark:bg-gray-700'
  };

  return (
    <button
      type="button"
      disabled={disabled}
      className={`relative inline-flex h-5 w-9 flex-shrink-0 rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none ${disabled ? '!cursor-not-allowed opacity-50' : '!cursor-pointer'} ${colorClasses[color]} ${className}`}
      role="switch"
      aria-checked={checked}
      aria-label={ariaLabel}
      onClick={() => !disabled && onChange(!checked)}
    >
      <span
        aria-hidden="true"
        className={`pointer-events-none inline-block h-4 w-4 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out ${checked ? 'translate-x-4' : 'translate-x-0'
          }`}
      />
    </button>
  );
}