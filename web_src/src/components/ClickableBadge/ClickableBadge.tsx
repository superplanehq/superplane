import React from 'react';
import { Badge } from '@/components/Badge/badge';
import { MaterialSymbol } from '@/components/MaterialSymbol/material-symbol';
import { ResourceLinkConfig } from '@/utils/resourceLinks';

interface ClickableBadgeProps {
  icon: string;
  color?: 'zinc' | 'blue' | 'green' | 'yellow' | 'red' | 'indigo' | 'gray';
  truncate?: boolean;
  className?: string;
  title?: string;
  children: React.ReactNode;
  resourceLinks?: ResourceLinkConfig[];
  badgeText?: string;
}

export function ClickableBadge({
  icon,
  color = 'zinc',
  truncate = false,
  className = '',
  title,
  children,
  resourceLinks = [],
  badgeText
}: ClickableBadgeProps) {
  const getRelevantLink = (): ResourceLinkConfig | null => {
    if (resourceLinks.length === 0) return null;

    const text = badgeText || children?.toString() || '';


    if (icon === 'assignment') {
      const repoLink = resourceLinks.find(link =>
        link.tooltip.includes('repository') || link.tooltip.includes('project')
      );
      return repoLink || resourceLinks[0];
    }


    if (icon === 'code') {
      const fileLink = resourceLinks.find(link =>
        link.tooltip.includes('pipeline file') ||
        link.tooltip.includes('workflow file') ||
        link.tooltip.includes(text)
      );
      return fileLink || null;
    }


    if (icon === 'graph_1') {
      const branchLink = resourceLinks.find(link =>
        link.tooltip.includes('at') && link.tooltip.includes(text)
      );
      return branchLink || resourceLinks[0];
    }


    return resourceLinks[0];
  };

  const handleClick = (e: React.MouseEvent) => {
    e.stopPropagation();

    const relevantLink = getRelevantLink();
    if (relevantLink) {
      window.open(relevantLink.url, '_blank', 'noopener,noreferrer');
    }
  };

  const relevantLink = getRelevantLink();
  const hasClickableLink = relevantLink !== null;
  const badgeTitle = relevantLink?.tooltip || title;

  return (
    <div
      className={`inline-flex items-center ${hasClickableLink ? 'cursor-pointer hover:opacity-80 transition-opacity' : ''} ${className}`}
      onClick={hasClickableLink ? handleClick : undefined}
      title={badgeTitle}
    >
      <Badge
        color={color}
        icon={icon}
        truncate={truncate}
        className="relative max-w-full"
      >
        <div className="flex items-center gap-1 min-w-0">
          <span className={truncate ? 'truncate' : ''}>{children}</span>
          {hasClickableLink && (
            <MaterialSymbol
              name="open_in_new"
              size="sm"
              className="text-zinc-500 dark:text-zinc-400 flex-shrink-0"
            />
          )}
        </div>
      </Badge>
    </div>
  );
}