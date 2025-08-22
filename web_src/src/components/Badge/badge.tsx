import React from 'react';
import { MaterialSymbol } from '../MaterialSymbol/material-symbol';

interface BadgeProps {
  children: React.ReactNode;
  color?: 'indigo' | 'gray' | 'blue' | 'red' | 'green' | 'yellow' | 'zinc';
  className?: string;
  icon?: string;
  truncate?: boolean;
  title?: string;
}

export function Badge({ children, color = 'gray', className = '', icon, truncate = false, title }: BadgeProps) {
  const colorClasses = {
    indigo: 'bg-indigo-100 text-indigo-800 dark:bg-indigo-900/20 dark:text-indigo-400',
    gray: 'bg-gray-100 text-gray-800 dark:bg-gray-900/20 dark:text-gray-400',
    blue: 'bg-blue-100 text-blue-800 dark:bg-blue-900/20 dark:text-blue-400',
    red: 'bg-red-100 text-red-800 dark:bg-red-900/20 dark:text-red-400',
    green: 'bg-green-100 text-green-800 dark:bg-green-900/20 dark:text-green-400',
    yellow: 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/20 dark:text-yellow-400',
    zinc: 'bg-zinc-600/10 text-zinc-700 hover:bg-zinc-600/30 transition-colors dark:bg-white/5 dark:text-zinc-400 dark:hover:bg-white/20'
  };

  return (

    <span title={title} className={`inline-flex items-center gap-x-1.5 px-1.5 py-0.5 rounded-md text-sm/5 font-medium sm:text-xs/5 forced-colors:outline ${colorClasses[color]} ${className}`}>
      {icon && <MaterialSymbol name={icon} size="md" className="flex-shrink-0" />}
      <span className={truncate ? 'truncate' : ''}>{children}</span>
    </span>
  );
}
