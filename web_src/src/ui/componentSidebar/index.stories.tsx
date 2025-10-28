import type { Meta, StoryObj } from '@storybook/react';
import { ComponentSidebar } from './';
import GithubIcon from "@/assets/icons/integrations/github.svg"
import { useState } from 'react';

const meta: Meta<typeof ComponentSidebar> = {
  title: 'ui/ComponentSidebar',
  component: ComponentSidebar,
  parameters: {
    layout: 'centered',
  },
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<typeof meta>;

const mockMetadata = [
  {
    icon: "book",
    label: "monarch-app",
  },
  {
    icon: "filter",
    label: "branch=main",
  },
];

const mockLatestEvents = [
  {
    title: "New commit",
    subtitle: "4m",
    state: "processed" as const,
    isOpen: false,
    receivedAt: new Date(),
    childEventsInfo: {
      count: 1,
      state: "processed" as const,
      waitingInfos: [],
    },
  },
  {
    title: "Pull request merged",
    subtitle: "3h",
    state: "discarded" as const,
    isOpen: false,
    receivedAt: new Date(Date.now() - 1000 * 60 * 30),
    values: {
      "Author": "Pedro Forestileao",
      "Commit": "feat: update component sidebar",
      "Branch": "feature/ui-update",
      "Type": "merge",
      "Event ID": "abc123-def456-ghi789",
    },
    childEventsInfo: {
      count: 3,
      state: "processed" as const,
      waitingInfos: [
        {
          icon: "check",
          info: "Tests passed",
        },
        {
          icon: "check",
          info: "Deploy completed",
        },
      ],
    },
  },
];

const mockNextInQueueEvents = [
  {
    title: "Deploy to staging",
    state: "waiting" as const,
    isOpen: false,
    receivedAt: new Date(Date.now() + 1000 * 60 * 5),
    childEventsInfo: {
      count: 2,
      state: "waiting" as const,
      waitingInfos: [
        {
          icon: "clock",
          info: "Waiting for approval",
          futureTimeDate: new Date(Date.now() + 1000 * 60 * 15),
        },
      ],
    },
  },
  {
    title: "Security scan",
    state: "waiting" as const,
    isOpen: false,
    receivedAt: new Date(Date.now() + 1000 * 60 * 10),
    childEventsInfo: {
      count: 1,
      state: "waiting" as const,
      waitingInfos: [],
    },
  },
];

export const Default: Story = {
  render: (args) => {
    const [latestEvents, setLatestEvents] = useState(mockLatestEvents);
    const [nextEvents, setNextEvents] = useState(mockNextInQueueEvents);

    const handleEventClick = (clickedEvent: any) => {
      console.log("Event clicked", clickedEvent);

      setLatestEvents(prev => prev.map(event =>
        event.title === clickedEvent.title && event.subtitle === clickedEvent.subtitle
          ? { ...event, isOpen: !event.isOpen }
          : event
      ));

      setNextEvents(prev => prev.map(event =>
        event.title === clickedEvent.title && event.subtitle === clickedEvent.subtitle
          ? { ...event, isOpen: !event.isOpen }
          : event
      ));
    };

    return (
      <div className="relative w-[32rem] h-[40rem]">
        <ComponentSidebar
          {...args}
          isOpen={true}
          latestEvents={latestEvents}
          nextInQueueEvents={nextEvents}
          onEventClick={handleEventClick}
          moreInQueueCount={2}
          onSeeFullHistory={() => console.log("See full history")}
        />
      </div>
    );
  },
  args: {
    metadata: mockMetadata,
    title: "Listen to code changes",
    iconSrc: GithubIcon,
    iconBackground: "bg-black",
    onExpandChildEvents: (childEventsInfo) => console.log("Expand child events", childEventsInfo),
    onReRunChildEvents: (childEventsInfo) => console.log("Re-run child events", childEventsInfo),
    onClose: () => console.log("Close sidebar"),
    onRun: () => console.log("Run action"),
    onDuplicate: () => console.log("Duplicate action"),
    onDocs: () => console.log("Documentation action"),
    onToggleView: () => console.log("Toggle view action"),
    onDeactivate: () => console.log("Deactivate action"),
    onDelete: () => console.log("Delete action"),
  },
};

export const WithInteractiveEvents: Story = {
  render: (args) => {
    const [latestEvents, setLatestEvents] = useState(mockLatestEvents);
    const [nextEvents, setNextEvents] = useState(mockNextInQueueEvents);

    const handleEventClick = (clickedEvent: any) => {
      console.log("Event clicked", clickedEvent);

      // Toggle isOpen state for latest events
      setLatestEvents(prev => prev.map(event =>
        event.title === clickedEvent.title && event.subtitle === clickedEvent.subtitle
          ? { ...event, isOpen: !event.isOpen }
          : event
      ));

      // Toggle isOpen state for next events
      setNextEvents(prev => prev.map(event =>
        event.title === clickedEvent.title && event.subtitle === clickedEvent.subtitle
          ? { ...event, isOpen: !event.isOpen }
          : event
      ));
    };

    return (
      <div className="relative w-[32rem] h-[40rem]">
        <ComponentSidebar
          {...args}
          isOpen={true}
          latestEvents={latestEvents}
          nextInQueueEvents={nextEvents}
          onEventClick={handleEventClick}
          moreInQueueCount={2}
          onSeeFullHistory={() => console.log("See full history")}
        />
      </div>
    );
  },
  args: {
    metadata: mockMetadata,
    title: "Interactive Event Sidebar",
    iconSrc: GithubIcon,
    iconBackground: "bg-black",
    onExpandChildEvents: (childEventsInfo) => console.log("Expand child events", childEventsInfo),
    onReRunChildEvents: (childEventsInfo) => console.log("Re-run child events", childEventsInfo),
    onClose: () => console.log("Close sidebar"),
    onRun: () => console.log("Run action"),
    onDuplicate: () => console.log("Duplicate action"),
    onDocs: () => console.log("Documentation action"),
    onToggleView: () => console.log("Toggle view action"),
  },
};

export const WithDifferentIcon: Story = {
  render: (args) => {
    const [latestEvents, setLatestEvents] = useState(mockLatestEvents);
    const [nextEvents, setNextEvents] = useState(mockNextInQueueEvents);

    const handleEventClick = (clickedEvent: any) => {
      console.log("Event clicked", clickedEvent);

      setLatestEvents(prev => prev.map(event =>
        event.title === clickedEvent.title && event.subtitle === clickedEvent.subtitle
          ? { ...event, isOpen: !event.isOpen }
          : event
      ));

      setNextEvents(prev => prev.map(event =>
        event.title === clickedEvent.title && event.subtitle === clickedEvent.subtitle
          ? { ...event, isOpen: !event.isOpen }
          : event
      ));
    };

    return (
      <div className="relative w-[32rem] h-[40rem]">
        <ComponentSidebar
          {...args}
          isOpen={true}
          latestEvents={latestEvents}
          nextInQueueEvents={nextEvents}
          onEventClick={handleEventClick}
          moreInQueueCount={2}
          onSeeFullHistory={() => console.log("See full history")}
        />
      </div>
    );
  },
  args: {
    metadata: [
      {
        icon: "database",
        label: "api-service",
      },
      {
        icon: "filter",
        label: "env=production",
      },
    ],
    title: "Database Changes",
    iconSlug: "database",
    iconColor: "text-blue-500",
    iconBackground: "bg-blue-200",
    onExpandChildEvents: (childEventsInfo) => console.log("Expand child events", childEventsInfo),
    onReRunChildEvents: (childEventsInfo) => console.log("Re-run child events", childEventsInfo),
    onClose: () => console.log("Close sidebar"),
    onRun: () => console.log("Run action"),
    onDeactivate: () => console.log("Deactivate action"),
    onDelete: () => console.log("Delete action"),
  },
};

export const ExtendedMetadata: Story = {
  render: (args) => {
    const [latestEvents, setLatestEvents] = useState(mockLatestEvents);
    const [nextEvents, setNextEvents] = useState(mockNextInQueueEvents);

    const handleEventClick = (clickedEvent: any) => {
      console.log("Event clicked", clickedEvent);

      setLatestEvents(prev => prev.map(event =>
        event.title === clickedEvent.title && event.subtitle === clickedEvent.subtitle
          ? { ...event, isOpen: !event.isOpen }
          : event
      ));

      setNextEvents(prev => prev.map(event =>
        event.title === clickedEvent.title && event.subtitle === clickedEvent.subtitle
          ? { ...event, isOpen: !event.isOpen }
          : event
      ));
    };

    return (
      <div className="relative w-[32rem] h-[40rem]">
        <ComponentSidebar
          {...args}
          isOpen={true}
          latestEvents={latestEvents}
          nextInQueueEvents={nextEvents}
          onEventClick={handleEventClick}
          moreInQueueCount={2}
          onSeeFullHistory={() => console.log("See full history")}
        />
      </div>
    );
  },
  args: {
    metadata: [
      {
        icon: "book",
        label: "large-enterprise-app",
      },
      {
        icon: "filter",
        label: "branch=main",
      },
      {
        icon: "tag",
        label: "v2.1.0",
      },
      {
        icon: "users",
        label: "team=backend",
      },
    ],
    title: "Enterprise Application Monitoring",
    iconSlug: "github",
    iconColor: "text-purple-500",
    iconBackground: "bg-purple-200",
    onExpandChildEvents: (childEventsInfo) => console.log("Expand child events", childEventsInfo),
    onReRunChildEvents: (childEventsInfo) => console.log("Re-run child events", childEventsInfo),
    onClose: () => console.log("Close sidebar"),
    onRun: () => console.log("Run action"),
    onDuplicate: () => console.log("Duplicate action"),
    onDocs: () => console.log("Documentation action"),
    onToggleView: () => console.log("Toggle view action"),
    onDeactivate: () => console.log("Deactivate action"),
    onDelete: () => console.log("Delete action"),
  },
};

export const ZeroState: Story = {
  render: (args) => {
    return (
      <div className="relative w-[32rem] h-[40rem]">
        <ComponentSidebar
          {...args}
          isOpen={true}
          latestEvents={[]}
          nextInQueueEvents={[]}
          onEventClick={() => console.log("Event clicked")}
          moreInQueueCount={0}
          onSeeFullHistory={() => console.log("See full history")}
        />
      </div>
    );
  },
  args: {
    metadata: mockMetadata,
    title: "Empty Component",
    iconSlug: "circle-dashed",
    iconColor: "text-gray-500",
    iconBackground: "bg-gray-200",
    onExpandChildEvents: (childEventsInfo) => console.log("Expand child events", childEventsInfo),
    onReRunChildEvents: (childEventsInfo) => console.log("Re-run child events", childEventsInfo),
    onClose: () => console.log("Close sidebar"),
  },
};

export const WithActionsDropdown: Story = {
  render: (args) => {
    return (
      <div className="relative w-[32rem] h-[40rem]">
        <ComponentSidebar
          {...args}
          isOpen={true}
          latestEvents={mockLatestEvents}
          nextInQueueEvents={mockNextInQueueEvents}
          onEventClick={() => console.log("Event clicked")}
          moreInQueueCount={3}
          onSeeFullHistory={() => console.log("See full history")}
        />
      </div>
    );
  },
  args: {
    metadata: mockMetadata,
    title: "Component with All Actions",
    iconSrc: GithubIcon,
    iconBackground: "bg-green-600",
    onExpandChildEvents: (childEventsInfo) => console.log("Expand child events", childEventsInfo),
    onReRunChildEvents: (childEventsInfo) => console.log("Re-run child events", childEventsInfo),
    onClose: () => console.log("Close sidebar"),
    onRun: () => {
      console.log("Run action triggered");
    },
    onDuplicate: () => {
      console.log("Duplicate action triggered");
    },
    onDocs: () => {
      console.log("Documentation action triggered");
    },
    onToggleView: () => {
      console.log("Toggle view action triggered");
    },
    onDeactivate: () => {
      console.log("Deactivate action triggered");
    },
    onDelete: () => {
      console.log("Delete action triggered");
    },
  },
};