interface FieldProps {
  children: React.ReactNode;
  className?: string;
}

export function Field({ children, className = "" }: FieldProps) {
  return <div className={`space-y-1 ${className}`}>{children}</div>;
}