import React from 'react';
import { AccordionItem } from '../AccordionItem';
import { RevertButton } from '../RevertButton';

interface EditableAccordionSectionProps {
  id: string;
  title: string;
  isOpen: boolean;
  onToggle: (sectionId: string) => void;
  isModified: boolean;
  onRevert: (sectionId: string) => void;
  count?: number;
  countLabel?: string;
  requiredBadge?: boolean;
  children: React.ReactNode;
  validationError?: string;
  className?: string;
  hasError?: boolean;
}

export function EditableAccordionSection({
  id,
  title,
  isOpen,
  onToggle,
  isModified,
  onRevert,
  count,
  countLabel,
  requiredBadge = false,
  children,
  validationError,
  className,
  hasError = false
}: EditableAccordionSectionProps) {
  const titleContent = (
    <div className="flex items-center justify-between w-full">
      <div className="flex items-center gap-2">
        <span className={`text-sm ${hasError ? 'text-red-600 dark:text-red-400' : 'text-zinc-600 dark:text-zinc-100'}`}>{title}</span>
        <RevertButton
          sectionId={id}
          isModified={isModified}
          onRevert={onRevert}
        />
      </div>
      <div className="flex items-center gap-2">
        {count !== undefined && count > 0 && (
          <span className="text-xs text-zinc-600 dark:text-zinc-400 font-normal pr-2">
            {count} {countLabel || 'items'}
          </span>
        )}
        {requiredBadge && (
          <span className="text-xs text-blue-600 font-medium">Required</span>
        )}
      </div>
    </div>
  );

  return (
    <AccordionItem
      id={id}
      title={titleContent}
      isOpen={isOpen}
      onToggle={onToggle}
      className={className}
    >
      <div className="space-y-2">
        {validationError && (
          <div className="text-xs text-red-600 mb-2">
            {validationError}
          </div>
        )}
        {children}
      </div>
    </AccordionItem>
  );
}