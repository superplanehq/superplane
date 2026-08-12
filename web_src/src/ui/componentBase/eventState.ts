import type React from "react";

export type EventState = "success" | "failed" | "neutral" | "queued" | "running" | string;

export interface EventStateStyle {
  icon: string;
  textColor: string;
  backgroundColor: string;
  badgeColor: string;
  label?: string;
}

export type EventStateMap = Record<EventState, EventStateStyle>;

export interface EventSection {
  showAutomaticTime?: boolean;
  receivedAt?: Date;
  eventId: string;
  eventState?: EventState;
  eventTitle?: string;
  eventSubtitle?: string | React.ReactNode;
  handleComponent?: React.ReactNode;
}
