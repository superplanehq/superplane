import { ReactNode, useRef } from 'react';
import Tippy from '@tippyjs/react';
import { Instance } from 'tippy.js';
import 'tippy.js/dist/tippy.css';
import './persistent-tooltip.css';
import { MaterialSymbol } from '@/components/MaterialSymbol/material-symbol';

interface PersistentTooltipProps {
  children: ReactNode;
  content: ReactNode;
  title?: string;
  maxWidth?: number;
  maxHeight?: string;
  placement?: 'top' | 'bottom' | 'left' | 'right' | 'top-start' | 'top-end' | 'bottom-start' | 'bottom-end' | 'left-start' | 'left-end' | 'right-start' | 'right-end';
  trigger?: string;
  showDismissButton?: boolean;
  className?: string;
}

export function PersistentTooltip({
  children,
  content,
  title,
  maxWidth = 600,
  maxHeight = 'max-h-96',
  placement = 'bottom-start',
  trigger = 'click',
  showDismissButton = true,
  className = ''
}: PersistentTooltipProps) {
  const tippyRef = useRef<Instance | null>(null);

  const tooltipContent = (
    <div className={`bg-white dark:bg-zinc-800 border border-zinc-200 dark:border-zinc-700 rounded-lg shadow-lg p-4 max-w-2xl ${maxHeight} overflow-y-auto ${className}`}>
      {(title || showDismissButton) && (
        <div className="flex justify-between items-start mb-3">
          {title && (
            <h3 className="text-sm font-semibold text-zinc-900 dark:text-zinc-100">
              {title}
            </h3>
          )}
          {showDismissButton && (
            <button
              onClick={(e) => {
                e.stopPropagation();
                e.preventDefault();
                if (tippyRef.current) {
                  tippyRef.current.hide();
                }
              }}
              className="text-zinc-400 hover:text-zinc-600 dark:hover:text-zinc-300 ml-2 flex-shrink-0"
            >
              <MaterialSymbol name="close" size="sm" />
            </button>
          )}
        </div>
      )}
      <div className="prose prose-sm dark:prose-invert text-zinc-700 dark:text-zinc-300">
        {content}
      </div>
    </div>
  );

  return (
    <Tippy
      content={tooltipContent}
      interactive={true}
      trigger={trigger}
      placement={placement}
      hideOnClick={false}
      arrow={true}
      theme="light-border"
      maxWidth={maxWidth}
      zIndex={9999}
      onCreate={(instance) => {
        tippyRef.current = instance;
      }}
    >
      <span className="inline-block">{children}</span>
    </Tippy>
  );
}