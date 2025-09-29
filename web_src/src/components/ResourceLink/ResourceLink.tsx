import { MaterialSymbol } from '@/components/MaterialSymbol/material-symbol';
import { ResourceLinkConfig } from '@/utils/resourceLinks';

interface ResourceLinkProps {
  config: ResourceLinkConfig;
  className?: string;
  variant?: 'standalone' | 'badge';
}

export function ResourceLink({ config, className = '', variant = 'standalone' }: ResourceLinkProps) {
  const handleClick = (e: React.MouseEvent) => {
    e.stopPropagation();
    window.open(config.url, '_blank', 'noopener,noreferrer');
  };

  const iconColorClass = variant === 'badge'
    ? 'text-zinc-500 dark:text-zinc-400 hover:text-zinc-700 dark:hover:text-zinc-200'
    : 'text-blue-600 dark:text-blue-400 hover:text-blue-800 dark:hover:text-blue-300';

  return (
    <button
      onClick={handleClick}
      title={config.tooltip}
      className={`inline-flex items-center gap-1 transition-colors ${iconColorClass} ${className}`}
    >
      <MaterialSymbol name="open_in_new" size="sm" />
    </button>
  );
}

interface ResourceLinksProps {
  links: ResourceLinkConfig[];
  className?: string;
  variant?: 'standalone' | 'badge';
}

export function ResourceLinks({ links, className = '', variant = 'standalone' }: ResourceLinksProps) {
  if (links.length === 0) return null;

  return (
    <div className={`inline-flex items-center gap-1 ${className}`}>
      {links.map((link, index) => (
        <ResourceLink key={index} config={link} variant={variant} />
      ))}
    </div>
  );
}