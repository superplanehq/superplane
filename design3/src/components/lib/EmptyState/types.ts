export interface EmptyStateAction {
  label: string;
  onClick?: () => void;
  href?: string;
}

export interface EmptyStatePrimaryAction extends EmptyStateAction {
  color?: 'blue' | 'red' | 'green' | 'yellow' | 'zinc';
}

export type EmptyStateSize = 'sm' | 'md' | 'lg';

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
  primaryAction?: EmptyStatePrimaryAction;
  /** Secondary call to action link */
  secondaryAction?: EmptyStateAction;
  /** Additional CSS classes */
  className?: string;
  /** Size variant for spacing */
  size?: EmptyStateSize;
}