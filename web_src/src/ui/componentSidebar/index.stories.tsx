import type { Meta, StoryObj } from "@storybook/react";
import { ComponentSidebar } from "./";
import GithubIcon from "@/assets/icons/integrations/github.svg";
import { useState } from "react";
import { MemoryRouter } from "react-router-dom";
import { DEFAULT_EVENT_STATE_MAP } from "../componentBase";

const meta: Meta<typeof ComponentSidebar> = {
  title: "ui/ComponentSidebar",
  component: ComponentSidebar,
  parameters: {
    layout: "centered",
  },
  tags: ["autodocs"],
  decorators: [
    (Story) => (
      <MemoryRouter>
        <Story />
      </MemoryRouter>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof meta>;

const mockLatestEvents = [
  {
    id: "event-1",
    title: "New commit",
    subtitle: "4m",
    state: "success" as const,
    isOpen: false,
    receivedAt: new Date(),
  },
  {
    id: "event-2",
    title: "Pull request merged",
    subtitle: "3h",
    state: "discarded" as const,
    isOpen: false,
    receivedAt: new Date(Date.now() - 1000 * 60 * 30),
    values: {
      Author: "Pedro Forestileao",
      Commit: "feat: update component sidebar",
      Branch: "feature/ui-update",
      Type: "merge",
      "Event ID": "abc123-def456-ghi789",
    },
  },
];

const mockNextInQueueEvents = [
  {
    id: "queue-1",
    title: "Deploy to staging",
    subtitle: "5m",
    state: "waiting" as const,
    isOpen: false,
    receivedAt: new Date(Date.now() + 1000 * 60 * 5),
  },
  {
    id: "queue-2",
    title: "Security scan",
    subtitle: "10m",
    state: "waiting" as const,
    isOpen: false,
    receivedAt: new Date(Date.now() + 1000 * 60 * 10),
  },
];

const mockAllHistoryEvents = [
  {
    id: "history-1",
    title: "Initial commit",
    subtitle: "2d",
    state: "success" as const,
    isOpen: false,
    receivedAt: new Date(Date.now() - 1000 * 60 * 60 * 48),
    values: {
      Author: "John Doe",
      Commit: "chore: initial project setup",
      Branch: "main",
      Type: "commit",
      "Event ID": "init123-setup456-base789",
    },
  },
  {
    id: "history-2",
    title: "Feature branch created",
    subtitle: "1d",
    state: "success" as const,
    isOpen: false,
    receivedAt: new Date(Date.now() - 1000 * 60 * 60 * 24),
    values: {
      Author: "Jane Smith",
      Branch: "feature/user-auth",
      Type: "branch_creation",
      "Event ID": "branch123-auth456-create789",
    },
  },
  {
    id: "history-3",
    title: "Tests passed",
    subtitle: "18h",
    state: "success" as const,
    isOpen: false,
    receivedAt: new Date(Date.now() - 1000 * 60 * 60 * 18),
    values: {
      "Test Suite": "Unit Tests",
      Coverage: "94%",
      Duration: "2m 14s",
      "Event ID": "test123-pass456-coverage789",
    },
  },
  {
    id: "history-4",
    title: "Deployment failed",
    subtitle: "12h",
    state: "discarded" as const,
    isOpen: false,
    receivedAt: new Date(Date.now() - 1000 * 60 * 60 * 12),
    values: {
      Environment: "staging",
      Error: "Port 3000 already in use",
      "Retry Count": "3",
      "Event ID": "deploy123-fail456-port789",
    },
  },
  {
    id: "history-5",
    title: "Code review submitted",
    subtitle: "8h",
    state: "success" as const,
    isOpen: false,
    receivedAt: new Date(Date.now() - 1000 * 60 * 60 * 8),
    values: {
      Reviewer: "Alex Johnson",
      "Files Changed": "12",
      Comments: "3",
      "Event ID": "review123-submit456-approved789",
    },
  },
  {
    id: "history-6",
    title: "Security audit completed",
    subtitle: "6h",
    state: "success" as const,
    isOpen: false,
    receivedAt: new Date(Date.now() - 1000 * 60 * 60 * 6),
    values: {
      "Vulnerabilities Found": "0",
      "Scan Duration": "5m 32s",
      "Tools Used": "Snyk, OWASP ZAP",
      "Event ID": "security123-audit456-clean789",
    },
  },
  {
    id: "history-7",
    title: "Build process started",
    subtitle: "5h",
    state: "running" as const,
    isOpen: false,
    receivedAt: new Date(Date.now() - 1000 * 60 * 60 * 5),
  },
  {
    id: "history-8",
    title: "Merge request created",
    subtitle: "4h",
    state: "success" as const,
    isOpen: false,
    receivedAt: new Date(Date.now() - 1000 * 60 * 60 * 4),
    values: {
      "Source Branch": "feature/ui-improvements",
      "Target Branch": "main",
      Author: "Sarah Wilson",
      "Event ID": "merge123-request456-ui789",
    },
  },
  ...mockLatestEvents,
  ...mockNextInQueueEvents,
];

export const Default: Story = {
  render: (args) => {
    const [latestEvents, setLatestEvents] = useState(mockLatestEvents);
    const [nextEvents, setNextEvents] = useState(mockNextInQueueEvents);

    const handleEventClick = (clickedEvent: any) => {
      console.log("Event clicked", clickedEvent);

      setLatestEvents((prev) =>
        prev.map((event) =>
          event.title === clickedEvent.title && event.subtitle === clickedEvent.subtitle
            ? { ...event, isOpen: !event.isOpen }
            : event,
        ),
      );

      setNextEvents((prev) =>
        prev.map((event) =>
          event.title === clickedEvent.title && event.subtitle === clickedEvent.subtitle
            ? { ...event, isOpen: !event.isOpen }
            : event,
        ),
      );
    };

    return (
      <div className="relative w-[32rem] h-[40rem]">
        <ComponentSidebar
          {...args}
          nodeId="node_123abc"
          isOpen={true}
          latestEvents={latestEvents}
          nextInQueueEvents={nextEvents}
          onEventClick={handleEventClick}
          totalInQueueCount={2}
          totalInHistoryCount={5}
          onSeeFullHistory={() => console.log("See full history")}
        />
      </div>
    );
  },
  args: {
    nodeName: "Listen to code changes",
    iconSrc: GithubIcon,
    iconBackground: "bg-white",
    onClose: () => console.log("Close sidebar"),
    onRun: () => console.log("Run action"),
    onDuplicate: () => console.log("Duplicate action"),
    onDocs: () => console.log("Documentation action"),
    onEdit: () => console.log("Edit action"),
    onToggleView: () => console.log("Toggle view action"),
    onDeactivate: () => console.log("Deactivate action"),
    onDelete: () => console.log("Delete action"),
    getExecutionState: () => ({
      map: DEFAULT_EVENT_STATE_MAP,
      state: "success" as const,
    }),
  },
};

export const WithInteractiveEvents: Story = {
  render: (args) => {
    const [latestEvents, setLatestEvents] = useState(mockLatestEvents);
    const [nextEvents, setNextEvents] = useState(mockNextInQueueEvents);

    const handleEventClick = (clickedEvent: any) => {
      console.log("Event clicked", clickedEvent);

      // Toggle isOpen state for latest events
      setLatestEvents((prev) =>
        prev.map((event) =>
          event.title === clickedEvent.title && event.subtitle === clickedEvent.subtitle
            ? { ...event, isOpen: !event.isOpen }
            : event,
        ),
      );

      // Toggle isOpen state for next events
      setNextEvents((prev) =>
        prev.map((event) =>
          event.title === clickedEvent.title && event.subtitle === clickedEvent.subtitle
            ? { ...event, isOpen: !event.isOpen }
            : event,
        ),
      );
    };

    return (
      <div className="relative w-[32rem] h-[40rem]">
        <ComponentSidebar
          {...args}
          nodeId="node_123abc"
          isOpen={true}
          latestEvents={latestEvents}
          nextInQueueEvents={nextEvents}
          onEventClick={handleEventClick}
          totalInQueueCount={2}
          totalInHistoryCount={3}
          onSeeFullHistory={() => console.log("See full history")}
        />
      </div>
    );
  },
  args: {
    nodeName: "Interactive Event Sidebar",
    iconSrc: GithubIcon,
    iconBackground: "bg-white",
    onClose: () => console.log("Close sidebar"),
    onRun: () => console.log("Run action"),
    onDuplicate: () => console.log("Duplicate action"),
    onDocs: () => console.log("Documentation action"),
    onToggleView: () => console.log("Toggle view action"),
    getExecutionState: () => ({
      map: DEFAULT_EVENT_STATE_MAP,
      state: "success" as const,
    }),
  },
};

export const WithDifferentIcon: Story = {
  render: (args) => {
    const [latestEvents, setLatestEvents] = useState(mockLatestEvents);
    const [nextEvents, setNextEvents] = useState(mockNextInQueueEvents);

    const handleEventClick = (clickedEvent: any) => {
      console.log("Event clicked", clickedEvent);

      setLatestEvents((prev) =>
        prev.map((event) =>
          event.title === clickedEvent.title && event.subtitle === clickedEvent.subtitle
            ? { ...event, isOpen: !event.isOpen }
            : event,
        ),
      );

      setNextEvents((prev) =>
        prev.map((event) =>
          event.title === clickedEvent.title && event.subtitle === clickedEvent.subtitle
            ? { ...event, isOpen: !event.isOpen }
            : event,
        ),
      );
    };

    return (
      <div className="relative w-[32rem] h-[40rem]">
        <ComponentSidebar
          {...args}
          nodeId="node_123abc"
          isOpen={true}
          latestEvents={latestEvents}
          nextInQueueEvents={nextEvents}
          onEventClick={handleEventClick}
          totalInQueueCount={2}
          totalInHistoryCount={4}
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
    nodeName: "Database Changes",
    iconSlug: "database",
    iconColor: "text-blue-700",
    iconBackground: "bg-blue-200",
    onClose: () => console.log("Close sidebar"),
    onRun: () => console.log("Run action"),
    onDeactivate: () => console.log("Deactivate action"),
    onDelete: () => console.log("Delete action"),
    getExecutionState: () => ({
      map: DEFAULT_EVENT_STATE_MAP,
      state: "success" as const,
    }),
  },
};

export const ExtendedMetadata: Story = {
  render: (args) => {
    const [latestEvents, setLatestEvents] = useState(mockLatestEvents);
    const [nextEvents, setNextEvents] = useState(mockNextInQueueEvents);

    const handleEventClick = (clickedEvent: any) => {
      console.log("Event clicked", clickedEvent);

      setLatestEvents((prev) =>
        prev.map((event) =>
          event.title === clickedEvent.title && event.subtitle === clickedEvent.subtitle
            ? { ...event, isOpen: !event.isOpen }
            : event,
        ),
      );

      setNextEvents((prev) =>
        prev.map((event) =>
          event.title === clickedEvent.title && event.subtitle === clickedEvent.subtitle
            ? { ...event, isOpen: !event.isOpen }
            : event,
        ),
      );
    };

    return (
      <div className="relative w-[32rem] h-[40rem]">
        <ComponentSidebar
          {...args}
          nodeId="node_123abc"
          isOpen={true}
          latestEvents={latestEvents}
          nextInQueueEvents={nextEvents}
          onEventClick={handleEventClick}
          totalInQueueCount={2}
          totalInHistoryCount={8}
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
    nodeName: "Enterprise Application Monitoring",
    iconSlug: "eye",
    iconColor: "text-purple-800",
    iconBackground: "bg-purple-300",
    onClose: () => console.log("Close sidebar"),
    onRun: () => console.log("Run action"),
    onDuplicate: () => console.log("Duplicate action"),
    onDocs: () => console.log("Documentation action"),
    onEdit: () => console.log("Edit action"),
    onToggleView: () => console.log("Toggle view action"),
    onDeactivate: () => console.log("Deactivate action"),
    onDelete: () => console.log("Delete action"),
    getExecutionState: () => ({
      map: DEFAULT_EVENT_STATE_MAP,
      state: "success" as const,
    }),
  },
};

export const ZeroState: Story = {
  render: (args) => {
    return (
      <div className="relative w-[32rem] h-[40rem]">
        <ComponentSidebar
          {...args}
          nodeId="node_123abc"
          isOpen={true}
          latestEvents={[]}
          nextInQueueEvents={[]}
          onEventClick={() => console.log("Event clicked")}
          totalInQueueCount={0}
          totalInHistoryCount={0}
          onSeeFullHistory={() => console.log("See full history")}
        />
      </div>
    );
  },
  args: {
    nodeName: "Empty Component",
    iconSlug: "circle-dashed",
    iconColor: "text-gray-800",
    iconBackground: "bg-gray-200",
    onClose: () => console.log("Close sidebar"),
    getExecutionState: () => ({
      map: DEFAULT_EVENT_STATE_MAP,
      state: "success" as const,
    }),
  },
};

export const WithActionsDropdown: Story = {
  render: (args) => {
    return (
      <div className="relative w-[32rem] h-[40rem]">
        <ComponentSidebar
          {...args}
          nodeId="node_123abc"
          isOpen={true}
          latestEvents={mockLatestEvents}
          nextInQueueEvents={mockNextInQueueEvents}
          onEventClick={() => console.log("Event clicked")}
          totalInQueueCount={3}
          totalInHistoryCount={7}
          onSeeFullHistory={() => console.log("See full history")}
        />
      </div>
    );
  },
  args: {
    nodeName: "Component with All Actions",
    iconSrc: GithubIcon,
    iconBackground: "bg-white",
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
    onEdit: () => {
      console.log("Edit action triggered");
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

export const WithFullHistory: Story = {
  render: (args) => {
    const [latestEvents, setLatestEvents] = useState(mockLatestEvents);
    const [nextEvents, setNextEvents] = useState(mockNextInQueueEvents);
    const [allEvents, setAllEvents] = useState(mockAllHistoryEvents);
    const [hasMore, setHasMore] = useState(true);
    const [loadingMore, setLoadingMore] = useState(false);

    const handleEventClick = (clickedEvent: any) => {
      console.log("Event clicked", clickedEvent);

      // Toggle isOpen state for latest events
      setLatestEvents((prev) =>
        prev.map((event) => (event.id === clickedEvent.id ? { ...event, isOpen: !event.isOpen } : event)),
      );

      // Toggle isOpen state for next events
      setNextEvents((prev) =>
        prev.map((event) => (event.id === clickedEvent.id ? { ...event, isOpen: !event.isOpen } : event)),
      );

      // Toggle isOpen state for all events
      setAllEvents((prev) =>
        prev.map((event) => (event.id === clickedEvent.id ? { ...event, isOpen: !event.isOpen } : event)),
      );
    };

    const handleLoadMore = () => {
      setLoadingMore(true);
      // Simulate loading more events
      setTimeout(() => {
        const moreEvents = [
          {
            id: "history-extra-1",
            title: "Database migration completed",
            subtitle: "3d",
            state: "success" as const,
            isOpen: false,
            receivedAt: new Date(Date.now() - 1000 * 60 * 60 * 72),
          },
          {
            id: "history-extra-2",
            title: "Backup created",
            subtitle: "4d",
            state: "success" as const,
            isOpen: false,
            receivedAt: new Date(Date.now() - 1000 * 60 * 60 * 96),
          },
        ];
        setAllEvents((prev) => [...prev, ...moreEvents]);
        setLoadingMore(false);
        setHasMore(false); // No more events after this
      }, 1500);
    };

    return (
      <div className="relative w-[32rem] h-[40rem]">
        <ComponentSidebar
          {...args}
          nodeId="node_123abc"
          isOpen={true}
          latestEvents={latestEvents}
          nextInQueueEvents={nextEvents}
          getAllHistoryEvents={() => allEvents}
          onEventClick={handleEventClick}
          totalInQueueCount={5}
          totalInHistoryCount={10}
          onSeeFullHistory={() => console.log("See full history triggered")}
          onLoadMoreHistory={handleLoadMore}
          getHasMoreHistory={() => hasMore}
          getLoadingMoreHistory={() => loadingMore}
        />
      </div>
    );
  },
  args: {
    nodeName: "Full History Demo",
    iconSrc: GithubIcon,
    iconBackground: "bg-white",
    onClose: () => console.log("Close sidebar"),
    onRun: () => console.log("Run action"),
    onDuplicate: () => console.log("Duplicate action"),
    onDocs: () => console.log("Documentation action"),
    onEdit: () => console.log("Edit action"),
    onToggleView: () => console.log("Toggle view action"),
    onDeactivate: () => console.log("Deactivate action"),
    onDelete: () => console.log("Delete action"),
    getExecutionState: () => ({
      map: DEFAULT_EVENT_STATE_MAP,
      state: "success" as const,
    }),
  },
};

export const HistoryCountDemo: Story = {
  render: (args) => {
    const [latestEvents, setLatestEvents] = useState(mockLatestEvents);
    const [nextEvents, setNextEvents] = useState(mockNextInQueueEvents);

    const handleEventClick = (clickedEvent: any) => {
      console.log("Event clicked", clickedEvent);

      setLatestEvents((prev) =>
        prev.map((event) => (event.id === clickedEvent.id ? { ...event, isOpen: !event.isOpen } : event)),
      );

      setNextEvents((prev) =>
        prev.map((event) => (event.id === clickedEvent.id ? { ...event, isOpen: !event.isOpen } : event)),
      );
    };

    return (
      <div className="relative w-[32rem] h-[40rem]">
        <ComponentSidebar
          {...args}
          nodeId="node_123abc"
          isOpen={true}
          latestEvents={latestEvents}
          nextInQueueEvents={nextEvents}
          onEventClick={handleEventClick}
          totalInQueueCount={3} // Shows "3 more in the queue"
          totalInHistoryCount={15} // Shows "See full history" with 15 more events
          onSeeFullHistory={() => console.log("See full history clicked - showing 15 more history events")}
        />
      </div>
    );
  },
  args: {
    nodeName: "History vs Queue Counts Demo",
    iconSrc: GithubIcon,
    iconBackground: "bg-white",
    onClose: () => console.log("Close sidebar"),
    onRun: () => console.log("Run action"),
  },
};
