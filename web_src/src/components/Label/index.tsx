interface LabelProps {
  children: React.ReactNode;
  htmlFor?: string;
  className?: string;
}

export function Label({ children, htmlFor, className = "" }: LabelProps) {
  return (
    <label htmlFor={htmlFor} className={`block text-sm font-medium text-zinc-700 dark:text-zinc-300 ${className}`}>
      {children}
    </label>
  );
}
