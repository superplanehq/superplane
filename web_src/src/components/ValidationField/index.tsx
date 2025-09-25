import React from 'react';
import { Field } from '../Field';
import { Label } from '../Label';

interface ValidationFieldProps {
  label: string | React.ReactNode;
  children: React.ReactNode;
  error?: string;
  required?: boolean;
  htmlFor?: string;
  className?: string;
}

export function ValidationField({
  label,
  children,
  error,
  required = false,
  htmlFor,
  className = ""
}: ValidationFieldProps) {
  return (
    <Field className={className}>
      <Label htmlFor={htmlFor}>
        {label}
        {required && <span className="text-red-500 ml-1">*</span>}
      </Label>
      {children}
      {error && (
        <div className="text-xs text-red-600 mt-1">
          {error}
        </div>
      )}
    </Field>
  );
}