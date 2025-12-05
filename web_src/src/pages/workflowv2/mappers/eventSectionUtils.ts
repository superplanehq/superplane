import { EventSection } from "@/ui/componentBase";

export type EventState = "success" | "failed" | "neutral" | "next-in-queue" | "running";

interface BaseEventSectionProps {
  title: string;
  eventTitle?: string;
  eventSubtitle?: string;
  subtitle?: string;
  showAutomaticTime?: boolean;
  receivedAt?: Date;
  handleComponent?: React.ReactNode;
}

/**
 * Creates an EventSection for a successful/completed state
 */
export function success(props: BaseEventSectionProps): EventSection {
  return {
    ...props,
    iconSlug: "check",
    textColor: "text-green-700",
    backgroundColor: "bg-green-200",
    iconColor: "text-green-600 bg-green-600",
    iconSize: 12,
    iconClassName: "text-white",
    inProgress: false,
  };
}

/**
 * Creates an EventSection for a failed state
 */
export function failed(props: BaseEventSectionProps): EventSection {
  return {
    ...props,
    iconSlug: "x",
    textColor: "text-red-700",
    backgroundColor: "bg-red-200",
    iconColor: "text-red-600 bg-red-600",
    iconSize: 12,
    iconClassName: "text-white",
    inProgress: false,
  };
}

/**
 * Creates an EventSection for a neutral/idle state
 */
export function neutral(props: BaseEventSectionProps): EventSection {
  return {
    ...props,
    iconSlug: "circle",
    textColor: "text-gray-500",
    backgroundColor: "bg-gray-100",
    iconColor: "text-gray-400 bg-gray-400",
    iconSize: 12,
    iconClassName: "text-white",
    inProgress: false,
  };
}

/**
 * Creates an EventSection for a running/in-progress state
 */
export function running(props: BaseEventSectionProps): EventSection {
  return {
    ...props,
    iconSlug: "refresh-cw",
    textColor: "text-blue-800",
    backgroundColor: "bg-sky-100",
    iconColor: "text-blue-800",
    iconSize: 16,
    iconClassName: "animate-spin",
    inProgress: true,
  };
}

/**
 * Creates an EventSection for a queued/waiting state
 */
export function inQueue(props: BaseEventSectionProps): EventSection {
  return {
    ...props,
    iconSlug: "circle-dashed",
    textColor: "text-gray-500",
    backgroundColor: "bg-gray-100",
    iconColor: "text-gray-500",
    iconSize: 16,
    iconClassName: "",
    inProgress: false,
  };
}
