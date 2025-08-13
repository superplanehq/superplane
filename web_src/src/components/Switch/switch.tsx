interface SwitchProps {
  checked: boolean;
  onChange: (checked: boolean) => void;
  color?: 'blue' | 'green' | 'indigo';
  className?: string;
  'aria-label'?: string;
}

export function Switch({
  checked,
  onChange,
  color = 'blue',
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
      className={`relative inline-flex h-5 w-9 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-blue-600 focus:ring-offset-2 ${colorClasses[color]} ${className}`}
      role="switch"
      aria-checked={checked}
      aria-label={ariaLabel}
      onClick={() => onChange(!checked)}
    >
      <span
        aria-hidden="true"
        className={`pointer-events-none inline-block h-4 w-4 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out ${checked ? 'translate-x-4' : 'translate-x-0'
          }`}
      />
    </button>
  );
}