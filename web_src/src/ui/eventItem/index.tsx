import * as React from "react";
import type { LucideProps } from "lucide-react";

import { cn, resolveIcon } from "@/lib/utils";

import { Badge, type BadgeProps } from "../badge";
import { Item, ItemActions, ItemContent, ItemMedia, ItemTitle } from "../item";

const STATUS_STYLES = {
  success: {
    background: "bg-emerald-100",
    text: "text-emerald-600",
    icon: "check-circle-2",
  },
  warning: {
    background: "bg-amber-100",
    text: "text-amber-600",
    icon: "alert-triangle",
  },
  error: {
    background: "bg-rose-100",
    text: "text-rose-600",
    icon: "x-circle",
  },
  info: {
    background: "bg-sky-100",
    text: "text-sky-600",
    icon: "info",
  },
} as const;

type StatusKey = keyof typeof STATUS_STYLES;

type IconComponent = React.ComponentType<LucideProps>;

interface EventItemBadge extends Omit<BadgeProps, "variant"> {
  label: string;
  icon?: string;
  variant?: BadgeProps["variant"];
}

export interface EventItemProps {
  status: StatusKey;
  title: string;
  badges?: EventItemBadge[];
  href?: string;
  timestamp?: React.ReactNode;
  statusIcon?: string;
  className?: string;
}

const EventItem: React.FC<EventItemProps> = ({ status, title, badges, href, timestamp, statusIcon, className }) => {
  const statusConfig = STATUS_STYLES[status];
  const StatusIcon = resolveIcon(statusIcon, statusConfig.icon) as IconComponent;

  const baseClassName = cn("w-full rounded-full transition-colors", statusConfig.background, className);

  const children = (
    <>
      <ItemMedia>
        <StatusIcon className={cn("size-6", statusConfig.text)} />
      </ItemMedia>
      <ItemContent className="min-w-0 flex-1 overflow-hidden">
        <ItemTitle
          className={cn(
            "flex min-w-0 items-center gap-2 text-sm font-medium leading-snug overflow-hidden w-full",
            statusConfig.text,
          )}
        >
          {badges?.length ? (
            <span className="flex shrink-0 gap-1">
              {badges.map(({ label, icon, variant, className: badgeClassName, ...badgeProps }) => {
                const BadgeIcon = icon ? (resolveIcon(icon, "bell") as IconComponent) : null;

                return (
                  <Badge
                    key={label}
                    variant={variant ?? "secondary"}
                    className={cn("flex items-center gap-1", badgeClassName)}
                    {...badgeProps}
                  >
                    {BadgeIcon ? <BadgeIcon className="size-3" /> : null}
                    {label}
                  </Badge>
                );
              })}
            </span>
          ) : null}
          <span className="min-w-0 flex-1 truncate text-sm font-medium leading-snug">{title}</span>
        </ItemTitle>
      </ItemContent>
      {timestamp ? (
        <ItemActions className="shrink-0 self-center pl-1 pr-2">
          <span className="text-xs text-muted-foreground">{timestamp}</span>
        </ItemActions>
      ) : null}
    </>
  );

  if (href) {
    return (
      <Item size="xs" className={cn(baseClassName, "flex-nowrap overflow-hidden")} asChild>
        <a href={href} className="flex w-full items-stretch gap-2 overflow-hidden min-w-0">
          {children}
        </a>
      </Item>
    );
  }

  return (
    <Item size="xs" className={cn(baseClassName, "flex-nowrap overflow-hidden min-w-0")}>
      {children}
    </Item>
  );
};

export { EventItem };
