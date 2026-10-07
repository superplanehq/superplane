import { Avatar } from "@/components/Avatar/avatar";
import type { OrgUserDisplay } from "@/lib/orgUserDisplay";
import { UNKNOWN_ORG_USER_NAME } from "@/lib/orgUserDisplay";
import { cn } from "@/lib/utils";

const avatarSizeClass = {
  xs: "size-5",
  sm: "size-6",
  md: "size-8",
} as const;

interface OrgUserReferenceProps {
  display: OrgUserDisplay | null;
  size?: keyof typeof avatarSizeClass;
  showName?: boolean;
  emphasizeName?: boolean;
  nameClassName?: string;
  className?: string;
}

export function OrgUserReference({
  display,
  size = "sm",
  showName = true,
  emphasizeName = false,
  nameClassName,
  className,
}: OrgUserReferenceProps) {
  const resolvedDisplay = display ?? {
    id: "unknown",
    name: UNKNOWN_ORG_USER_NAME,
    initials: "?",
  };

  return (
    <span className={cn("inline-flex min-w-0 items-center gap-1.5", className)}>
      <Avatar
        src={resolvedDisplay.avatarUrl}
        initials={resolvedDisplay.initials}
        alt={resolvedDisplay.name}
        className={avatarSizeClass[size]}
      />
      {showName ? (
        <span className={cn("truncate text-foreground", emphasizeName && "font-semibold", nameClassName)}>
          {resolvedDisplay.name}
        </span>
      ) : null}
    </span>
  );
}
