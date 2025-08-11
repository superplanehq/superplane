import React from 'react';

interface BadgeProps {
  children: React.ReactNode;
  color?: 'indigo' | 'gray' | 'blue' | 'red' | 'green' | 'yellow';
  className?: string;
}

export function Badge({ children, color = 'gray', className = '' }: BadgeProps) {
  const colorClasses = {
    indigo: 'bg-indigo-100 text-indigo-800 dark:bg-indigo-900/20 dark:text-indigo-400',
    gray: 'bg-gray-100 text-gray-800 dark:bg-gray-900/20 dark:text-gray-400',
    blue: 'bg-blue-100 text-blue-800 dark:bg-blue-900/20 dark:text-blue-400',
    red: 'bg-red-100 text-red-800 dark:bg-red-900/20 dark:text-red-400',
    green: 'bg-green-100 text-green-800 dark:bg-green-900/20 dark:text-green-400',
    yellow: 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/20 dark:text-yellow-400'
  };

  return (
    <span className={`inline-flex items-center px-2 py-1 rounded-[calc(var(--radius-lg)-1px)] text-xs font-medium ${colorClasses[color]} ${className}`}>
      {children}
    </span>
  );
}