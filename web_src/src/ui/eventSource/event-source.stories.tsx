import type { Meta, StoryObj } from "@storybook/react";
import type { ComponentProps } from "react";

import githubIcon from "@/assets/icons/integrations/github.svg";
import kubernetesIcon from "@/assets/icons/integrations/kubernetes.svg";
import pagerDutyIcon from "@/assets/icons/integrations/pagerduty.svg";
import dataDogIcon from "@/assets/icons/integrations/datadog.svg";
import scheduleIcon from "@/assets/icons/schedule.svg";

import { EventSource, type EventSourceProps } from "./index";

const TYPES = {
  github: {
    label: "GitHub",
    sectionTone: "bg-gray-100",
    badgeTone: "bg-gray-950",
    iconSrc: githubIcon,
    title: "GitHub Source",
    resource: {
      label: "superplane-ui",
      href: "https://github.com/superplanehq/superplane-ui",
      icon: "book-marked",
    },
    eventType: { label: "push" } as EventSourceProps["eventType"],
    filters: ["branch = main", "status = success"] as string[],
    events: [
      {
        status: "success",
        title: "fix: open rejected events tab when the sidebar is already open",
        timestamp: "2m ago",
        badges: [{ label: "push" }],
        href: "#",
      },
    ] satisfies EventSourceProps["events"],
    meta: undefined,
  },
  kubernetes: {
    label: "Kubernetes",
    sectionTone: "bg-blue-100",
    badgeTone: "bg-blue-500",
    iconSrc: kubernetesIcon,
    title: "Cluster Events",
    resource: {
      label: "prod-cluster",
      href: "https://k8s.superplane.dev",
      icon: "boxes",
    },
    eventType: { label: "crashLoopBackOff" } as EventSourceProps["eventType"],
    filters: ["namespace != stage", "kind == deployment"] as string[],
    events: [
      {
        status: "success",
        title: "traffic-split applied to gateway",
        timestamp: "Just now",
        badges: [{ label: "mesh" }],
      },
      {
        status: "warning",
        title: "pod restart threshold exceeded",
        timestamp: "3m ago",
        badges: [{ label: "alert", variant: "destructive" }],
      },
    ] satisfies EventSourceProps["events"],
    meta: undefined,
  },
  pagerduty: {
    label: "PagerDuty",
    sectionTone: "bg-green-100",
    badgeTone: "bg-green-500",
    iconSrc: pagerDutyIcon,
    title: "PagerDuty Incidents",
    resource: {
      label: "On-call schedule",
      href: "https://pagerduty.com",
      icon: "clipboard-clock",
    },
    eventType: { label: "incident" } as EventSourceProps["eventType"],
    filters: ["urgency = high"] as string[],
    events: [
      {
        status: "warning",
        title: "P1: Payment gateway outage",
        timestamp: "5m ago",
        badges: [{ label: "alert", variant: "destructive" }],
      },
    ] satisfies EventSourceProps["events"],
    meta: undefined,
  },
  datadog: {
    label: "DataDog",
    sectionTone: "bg-purple-100",
    badgeTone: "bg-purple-500",
    iconSrc: dataDogIcon,
    title: "DataDog Monitors",
    resource: {
      label: "APM metrics",
      href: "https://app.datadoghq.com",
      icon: "activity",
    },
    eventType: undefined,
    filters: [] as string[],
    events: [
      {
        status: "warning",
        title: "p99 latency above threshold",
        timestamp: "4m ago",
        badges: [{ label: "apm" }],
      },
      {
        status: "info",
        title: "cache hit rate recovered",
        timestamp: "8m ago",
        badges: [{ label: "cache" }],
      },
    ] satisfies EventSourceProps["events"],
    meta: undefined,
  },
  schedule: {
    label: "Schedule",
    sectionTone: "bg-gray-100",
    badgeTone: "bg-gray-950",
    iconSrc: scheduleIcon,
    title: "Daily Build",
    resource: {
      label: "Every Friday at 09:15 am ET",
      href: "#",
      icon: "calendar-clock",
    },
    eventType: undefined,
    filters: [] as string[],
    events: [
      {
        status: "success",
        title: "Aug 2, 2025, 09:15 am ET",
        timestamp: "3m ago",
        badges: [{ label: "schedule" }],
      },
    ] satisfies EventSourceProps["events"],
    meta: [
      {
        icon: "clock-arrow-down",
        label: "Next run:",
        value: "Aug 10, 2024, 09:15am ET",
      },
    ] satisfies EventSourceProps["meta"],
  },
} as const;

type TypeKey = keyof typeof TYPES;
type StoryArgs = ComponentProps<typeof EventSource> & {
  type: TypeKey;
};

const meta = {
  title: "ui/Event Source",
  component: EventSource,
  tags: ["autodocs"],
  parameters: {
    layout: "centered",
  },
  argTypes: {
    type: {
      control: { type: "select" },
      options: Object.keys(TYPES),
    },
    content: {
      control: { disable: true },
      table: { disable: true },
    },
    sectionTone: {
      control: { disable: true },
      table: { disable: true },
    },
    badgeTone: {
      control: { disable: true },
      table: { disable: true },
    },
    badgeImageSrc: {
      control: { disable: true },
      table: { disable: true },
    },
    badgeImageAlt: {
      control: { disable: true },
      table: { disable: true },
    },
    badgeLabel: {
      control: { disable: true },
      table: { disable: true },
    },
    className: {
      control: { type: "text" },
      table: { disable: true },
    },
    footerContent: {
      control: { disable: true },
      table: { disable: true },
    },
    events: {
      control: { type: "object" },
      table: { disable: true },
    },
    title: {
      control: { disable: true },
      table: { disable: true },
    },
    eventType: {
      control: { disable: true },
      table: { disable: true },
    },
    resource: {
      control: { disable: true },
      table: { disable: true },
    },
    filters: {
      control: { disable: true },
      table: { disable: true },
    },
    meta: {
      control: { disable: true },
      table: { disable: true },
    },
    collapsed: {
      control: { type: "boolean" },
    },
  },
  args: {
    type: "github" satisfies TypeKey,
    selected: true,
    collapsed: false,
  },
  render: (args) => {
    const { type, ...rest } = args as StoryArgs;
    const config = TYPES[type] ?? TYPES.github;

    return (
      <EventSource
        {...rest}
        title={rest.title ?? config.title}
        content=""
        sectionTone={config.sectionTone}
        badgeTone={config.badgeTone}
        badgeLabel={config.label}
        badgeImageSrc={config.iconSrc}
        resource={rest.resource ?? config.resource}
        eventType={rest.eventType ?? config.eventType}
        filters={rest.filters ?? config.filters}
        meta={rest.meta ?? config.meta}
        events={rest.events ?? config.events}
      />
    );
  },
} satisfies Meta<StoryArgs>;

export default meta;

type Story = StoryObj<StoryArgs>;

export const Default: Story = {};

export const ZeroState: Story = {
  args: {
    type: "github",
    events: [],
    selected: false,
  },
};

export const Collapsed: Story = {
  args: {
    type: "github",
    collapsed: true,
    selected: false,
  },
};
