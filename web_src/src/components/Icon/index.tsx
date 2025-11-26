import clsx from "clsx";
import {
  Plus,
  X,
  Search,
  MoreVertical,
  ChevronDown,
  ChevronUp,
  Check,
  Edit,
  Trash2,
  User,
  Users,
  Building,
  Shield,
  Settings,
  ArrowLeftRight,
  LogOut,
  Copy,
  Eye,
  EyeOff,
  AlertTriangle,
  AlertCircle,
  Loader2,
  Save,
  RefreshCw,
  Mail,
  UserPlus,
  UserMinus,
} from "lucide-react";

export interface IconProps {
  /** The name of the icon (Material Symbol name) */
  name: string;
  /** Size variant */
  size?: "sm" | "md" | "lg" | "xl" | "4xl";
  /** Additional CSS classes */
  className?: string;
  /** Data slot attribute for button styling */
  "data-slot"?: string;
}

// Mapping from Material Symbol names to Lucide components
const iconMap: Record<
  string,
  React.ComponentType<{ className?: string; "aria-hidden"?: boolean; "data-slot"?: string }>
> = {
  // Navigation & Actions
  close: X,
  add: Plus,
  search: Search,
  more_vert: MoreVertical,
  keyboard_arrow_down: ChevronDown,
  keyboard_arrow_up: ChevronUp,
  expand_more: ChevronDown,
  check: Check,
  edit: Edit,
  delete: Trash2,
  save: Save,
  refresh: RefreshCw,

  // People & Organization
  person: User,
  group: Users,
  business: Building,
  shield: Shield,
  person_add: UserPlus,
  person_remove: UserMinus,
  group_add: UserPlus, // Using UserPlus as closest equivalent

  // Communication
  mail: Mail,

  // Visibility & Copy
  visibility: Eye,
  visibility_off: EyeOff,
  content_copy: Copy,

  // Alerts & Status
  warning: AlertTriangle,
  error: AlertCircle,

  // Loading & Progress
  progress_activity: Loader2,
  hourglass_empty: Loader2, // Using Loader2 as closest equivalent

  // Settings & Tools
  settings: Settings,
  integration_instructions: Settings, // Using Settings as closest equivalent
  swap_horiz: ArrowLeftRight,
  logout: LogOut,
  select_all: Check, // Using Check as closest equivalent
};

export function Icon({ name, size = "md", className, "data-slot": dataSlot }: IconProps) {
  const IconComponent = iconMap[name];

  if (!IconComponent) {
    console.warn(`Icon "${name}" not found in icon mapping`);
    return null;
  }

  const sizeClasses = {
    sm: "h-4 w-4", // 16px
    md: "h-4 w-4", // 16px
    lg: "h-5 w-5", // 20px
    xl: "h-6 w-6", // 24px
    "4xl": "h-8 w-8", // 32px
  };

  return (
    <IconComponent
      className={clsx("flex-shrink-0", sizeClasses[size], className)}
      aria-hidden={true}
      data-slot={dataSlot}
    />
  );
}

// Export for backward compatibility during migration
export const MaterialSymbol = Icon;
export const MaterialSymbolFilled = Icon;
export const MaterialSymbolLight = Icon;
export const MaterialSymbolBold = Icon;
