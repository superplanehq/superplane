import * as React from "react"

import { cn } from "@/lib/utils"

import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "../card"
import { Button } from "../button"
import { Badge } from "../badge"
import { EventItem, type EventItemProps } from "../eventItem"
import {
  HoverCard,
  HoverCardContent,
  HoverCardTrigger,
} from "../hoverCard"
import type { LucideIcon } from "lucide-react"
import * as LucideIcons from "lucide-react"
import { BookMarked, Funnel } from "lucide-react"

export interface EventSourceProps {
  title: string
  content: string
  resource: {
    label: string
    href: string
    icon?: string
  }
  eventType?: {
    label: string
    onClick?: () => void
  }
  filters?: string[]
  events?: EventItemProps[]
  meta?: Array<{
    icon?: string
    label: string
    value: React.ReactNode
  }>
  sectionTone?: string
  badgeTone?: string
  badgeImageSrc?: string
  badgeImageAlt?: string
  badgeLabel?: string
  footerContent?: React.ReactNode
  className?: string
  selected?: boolean
  collapsed?: boolean
}

const DEFAULT_SECTION_TONE = "bg-gray-100"
const DEFAULT_BADGE_TONE = "bg-gray-950"

export const EventSource: React.FC<EventSourceProps> = ({
  title,
  content,
  resource,
  eventType,
  filters,
  events,
  meta,
  sectionTone = DEFAULT_SECTION_TONE,
  badgeTone = DEFAULT_BADGE_TONE,
  badgeImageSrc,
  badgeImageAlt,
  badgeLabel,
  className,
  selected = false,
  collapsed = false,
}) => {
  const resolveIcon = React.useCallback((slug?: string): LucideIcon => {
    if (!slug) {
      return BookMarked
    }

    const pascalCase = slug
      .split("-")
      .map((segment) => segment.charAt(0).toUpperCase() + segment.slice(1))
      .join("")

    const candidate = (LucideIcons as Record<string, unknown>)[pascalCase]

    if (
      candidate &&
      (typeof candidate === "function" ||
        (typeof candidate === "object" && "render" in candidate))
    ) {
      return candidate as LucideIcon
    }

    return BookMarked
  }, [])

  const ResourceIcon = React.useMemo(() => {
    if (!resource.icon) {
      return BookMarked
    }

    return resolveIcon(resource.icon)
  }, [resource.icon, resolveIcon])

  if (collapsed) {
    return (
      <div className={cn("flex w-fit flex-col items-center", className)}>
        <div
          className={cn(
            "flex h-20 w-20 items-center justify-center rounded-full text-white",
            badgeTone,
            selected ? "border-[3px] border-black" : "border border-border",
          )}
        >
          {badgeImageSrc ? (
            <img
              src={badgeImageSrc}
              alt={badgeImageAlt ?? ""}
              className="h-12 w-12 object-contain"
              draggable={false}
            />
          ) : (
            <ResourceIcon className="size-10" />
          )}
          {badgeLabel ? <span className="sr-only">{badgeLabel}</span> : null}
        </div>
        <CardTitle className="text-base font-semibold text-neutral-900 pt-1">
          {title}
        </CardTitle>
        <Button
          variant="linkSubdued"
          className="justify-center text-sm"
          asChild
        >
          <a
            href={resource.href}
            target="_blank"
            rel="noreferrer"
            className="flex items-center"
          >
            <ResourceIcon className="size-4" />
            {resource.label}
          </a>
        </Button>
      </div>
    )
  }

  return (
    <div className={cn("relative w-[26rem]", className)}>
      <div
        className={cn(
          "absolute left-0 top-0 flex h-14 w-14 -translate-x-1/2 -translate-y-1/2 items-center justify-center rounded-full text-white",
          badgeTone,
        )}
      >
        {badgeImageSrc ? (
          <img
            src={badgeImageSrc}
            alt={badgeImageAlt ?? ""}
            className="h-9 w-9 object-contain"
            draggable={false}
          />
        ) : null}
        {badgeLabel ? <span className="sr-only">{badgeLabel}</span> : null}
      </div>
      <Card
        className={cn(
          "flex h-full w-full flex-col overflow-hidden p-0",
          selected
            ? "border-[3px] border-black shadow-none"
            : "border border-border shadow-sm",
        )}
      >
        <CardHeader
          className={cn(
            "space-y-2 rounded-none pb-4 text-base text-neutral-900",
            sectionTone,
          )}
        >
          <CardTitle>{title}</CardTitle>
          <CardDescription className="flex flex-col items-start gap-2 py-2 text-sm text-neutral-900">
            <Button variant="linkSubdued" className="justify-start" asChild>
              <a
                href={resource.href}
                target="_blank"
                rel="noreferrer"
                  className="flex items-center gap-2"
                >
                  <ResourceIcon />
                  {resource.label}
                </a>
              </Button>
              {(eventType || (filters && filters.length > 0)) && (
                <div className="flex items-center gap-2">
                  {eventType ? (
                    <Button
                      variant="linkSubdued"
                      className="justify-start"
                      onClick={eventType.onClick}
                    >
                      <Funnel />
                      {eventType.label}
                    </Button>
                  ) : null}
                  {filters && filters.length > 0 ? (
                    <HoverCard openDelay={150} closeDelay={150}>
                      <HoverCardTrigger asChild>
                        <Badge variant="default" className="cursor-pointer">
                          +{filters.length} filters
                        </Badge>
                      </HoverCardTrigger>
                      <HoverCardContent className="w-56 space-y-2 text-xs">
                        <p className="text-sm font-medium text-neutral-900">
                          Active filters
                        </p>
                        <ul className="list-disc space-y-1 pl-5 text-muted-foreground">
                          {filters.map((filter) => (
                            <li key={filter}>{filter}</li>
                          ))}
                        </ul>
                      </HoverCardContent>
                    </HoverCard>
                  ) : null}
                </div>
              )}
              {meta && meta.length > 0 ? (
                <div className="mt-2 flex w-full flex-col gap-2 text-sm">
                  {meta.map(({ icon, label: metaLabel, value }, index) => {
                    const MetaIcon = resolveIcon(icon)

                    return (
                      <div
                        key={`${metaLabel}-${index}`}
                        className="flex items-center gap-2 text-muted-foreground"
                      >
                        <MetaIcon className="size-4 text-muted-foreground" />
                        <span className="text-sm text-muted-foreground font-medium">
                          {metaLabel}
                        </span>
                        <span className="text-muted-foreground font-medium">{value}</span>
                      </div>
                    )
                  })}
                </div>
              ) : null}
            </CardDescription>
        </CardHeader>
        {content ? (
          <CardContent
            className={cn(
              "rounded-none p-6 text-sm leading-relaxed text-neutral-900",
              sectionTone,
            )}
          >
            {content}
          </CardContent>
        ) : null}
        <CardFooter className="flex flex-col items-start gap-3 bg-white p-6 pt-4 text-neutral-900">
          <div className="text-sm font-semibold text-muted-foreground">
            {events && events.length === 1 ? "Latest Event" : "Latest Events"}
          </div>
          {events && events.length > 0 ? (
            <div className="flex w-full flex-col gap-2">
              {events.map((event, index) => (
                <EventItem
                  key={`${event.title}-${index}`}
                  {...event}
                />
              ))}
            </div>
          ) : (
            <p className="text-xs text-muted-foreground">No events yet.</p>
          )}
        </CardFooter>
      </Card>
    </div>
  )
}
