import React from 'react';
import { Button } from '../Button/button';
import { Link } from '../Link/link';
import { MaterialSymbol } from '../MaterialSymbol/material-symbol';
import { Heading } from '../Heading/heading';
import { Text } from '../Text/text';

export interface EmptyStateProps {
  /** Optional image element (can be img, svg, or MaterialSymbol) */
  image?: React.ReactNode;
  /** Optional icon name for MaterialSymbol when no custom image provided */
  icon?: string;
  /** Short, concise title - preferably written as positive statement */
  title: string;
  /** Body text explaining next action to populate space */
  body: string;
  /** Primary call to action button */
  primaryAction?: {
    label: string;
    onClick?: () => void;
    href?: string;
    color?: 'blue' | 'red' | 'green' | 'yellow' | 'zinc';
  };
  /** Secondary call to action link */
  secondaryAction?: {
    label: string;
    onClick?: () => void;
    href?: string;
  };
  /** Additional CSS classes */
  className?: string;
  /** Size variant for spacing */
  size?: 'sm' | 'md' | 'lg';
}

export function EmptyState({
  image,
  icon,
  title,
  body,
  primaryAction,
  secondaryAction,
  className = '',
  size = 'md'
}: EmptyStateProps) {
  const sizeClasses = {
    sm: 'py-8',
    md: 'py-12',
    lg: 'py-16'
  };

  const iconSizeClasses = {
    sm: 'text-4xl mb-3',
    md: 'text-5xl mb-4', 
    lg: 'text-6xl mb-6'
  };

  const titleSizeClasses = {
    sm: 'text-lg',
    md: 'text-xl',
    lg: 'text-2xl'
  };

  return (
    <div className={`text-center ${sizeClasses[size]} ${className}`}>
      {/* Image or Icon */}
      {image ? (
        <div className="flex justify-center mb-4">
          {image}
        </div>
      ) : (
        <MaterialSymbol 
          name={icon} 
          className={`mx-auto text-zinc-400 dark:text-zinc-500 ${iconSizeClasses[size]}`}
        />
      )}

      {/* Title */}
      <Heading 
        level={3} 
        className={`font-semibold text-zinc-900 dark:text-white mb-2 ${titleSizeClasses[size]}`}
      >
        {title}
      </Heading>

      {/* Body */}
      <Text className="text-zinc-600 dark:text-zinc-400 max-w-md mx-auto mb-6">
        {body}
      </Text>

      {/* Primary Action */}
      {primaryAction && (
        <div className="mb-4">
          {primaryAction.href ? (
            <Button 
              href={primaryAction.href}
              color={primaryAction.color || 'blue'}
              onClick={primaryAction.onClick}
            >
              {primaryAction.label}
            </Button>
          ) : (
            <Button 
              color={primaryAction.color || 'blue'}
              onClick={primaryAction.onClick}
            >
              {primaryAction.label}
            </Button>
          )}
        </div>
      )}

      {/* Secondary Action */}
      {secondaryAction && (
        <div>
          {secondaryAction.href ? (
            <Link 
              href={secondaryAction.href}
              className="text-sm text-blue-600 dark:text-blue-400 hover:text-blue-800 dark:hover:text-blue-300"
              onClick={secondaryAction.onClick}
            >
              {secondaryAction.label}
            </Link>
          ) : (
            <Link 
              href="#"
              className="text-sm text-blue-600 dark:text-blue-400 hover:text-blue-800 dark:hover:text-blue-300"
              onClick={secondaryAction.onClick}
            >
              {secondaryAction.label}
            </Link>
          )}
        </div>
      )}
    </div>
  );
}