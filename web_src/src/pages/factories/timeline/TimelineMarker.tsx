import { Avatar } from "@/components/Avatar/avatar";
import type { OrgUserDisplay } from "@/lib/orgUserDisplay";
import { UserRound, type LucideIcon } from "lucide-react";

interface TimelineMarkerProps {
  display?: OrgUserDisplay | null;
  icon?: LucideIcon;
}

export function TimelineMarker({ display, icon: Icon = UserRound }: TimelineMarkerProps) {
  if (display) {
    return (
      <span className="relative">
        <Avatar src={display.avatarUrl} initials={display.initials} alt={display.name} className="size-6" />
      </span>
    );
  }

  return (
    <span className="relative flex size-6 shrink-0 items-center justify-center rounded-full border border-border bg-background">
      <Icon className="size-3 text-muted-foreground" aria-hidden />
    </span>
  );
}
