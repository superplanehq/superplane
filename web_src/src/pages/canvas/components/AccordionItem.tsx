import { memo } from 'react';

interface AccordionItemProps {
  id: string;
  title: React.ReactNode;
  children: React.ReactNode;
  isOpen: boolean;
  onToggle: (id: string) => void;
}

export const AccordionItem = memo(function AccordionItem({ id, title, children, isOpen, onToggle }: AccordionItemProps) {
  return (
    <div className="border-b border-zinc-200 dark:border-zinc-700">
      <button
        className="w-full px-4 py-3 text-left flex justify-between items-center hover:bg-zinc-50 dark:hover:bg-zinc-800/50 transition-colors"
        onClick={() => onToggle(id)}
      >
        <div className="font-medium text-zinc-900 dark:text-zinc-100 w-full">{title}</div>
        <span className="material-symbols-outlined text-zinc-500 dark:text-zinc-400">
          {isOpen ? 'expand_less' : 'expand_more'}
        </span>
      </button>
      {isOpen && (
        <div className="px-4 pb-4">
          {children}
        </div>
      )}
    </div>
  );
});