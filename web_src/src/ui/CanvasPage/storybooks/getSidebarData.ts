import { CanvasNode, SidebarData, SidebarEvent } from "../index";

/**
 * Helper function to create fake sidebar data from node data for Storybook stories.
 * This extracts the relevant data from a node and formats it for the ComponentSidebar.
 */
export function createGetSidebarData(nodes: CanvasNode[]) {
  return (nodeId: string): SidebarData | null => {
    const node = nodes.find((n) => n.id === nodeId);
    if (!node) return null;

    const data = node.data as any;

    // Handle trigger nodes
    if (data.type === "trigger" && data.trigger) {
      const trigger = data.trigger;
      const latestEvents: SidebarEvent[] = [];

      if (trigger.lastEventData) {
        latestEvents.push({
          title: trigger.lastEventData.title,
          subtitle: trigger.lastEventData.subtitle,
          state: trigger.lastEventData.state || "processed",
          isOpen: false,
          receivedAt: trigger.lastEventData.receivedAt,
          values: trigger.lastEventData.values,
          childEventsInfo: trigger.lastEventData.childEventsInfo,
        });
      }

      return {
        title: trigger.title,
        iconSrc: trigger.iconSrc,
        iconSlug: trigger.iconSlug,
        iconColor: trigger.iconColor,
        iconBackground: trigger.iconBackground,
        metadata: trigger.metadata || [],
        latestEvents,
        nextInQueueEvents: [],
        moreInQueueCount: 0,
      };
    }

    // Handle composite nodes
    if (data.type === "composite" && data.composite) {
      const composite = data.composite;
      const latestEvents: SidebarEvent[] = [];
      const nextInQueueEvents: SidebarEvent[] = [];

      if (composite.lastRunItem) {
        latestEvents.push({
          title: composite.lastRunItem.title,
          subtitle: composite.lastRunItem.subtitle,
          state: composite.lastRunItem.state || "processed",
          isOpen: false,
          receivedAt: composite.lastRunItem.receivedAt,
          values: composite.lastRunItem.values,
          childEventsInfo: composite.lastRunItem.childEventsInfo,
        });
      }

      if (composite.nextInQueue) {
        nextInQueueEvents.push({
          title: composite.nextInQueue.title,
          subtitle: composite.nextInQueue.subtitle,
          state: "waiting",
          isOpen: false,
          receivedAt: composite.nextInQueue.receivedAt,
        });
      }

      return {
        title: composite.title,
        iconSrc: composite.iconSrc,
        iconSlug: composite.iconSlug,
        iconColor: composite.iconColor,
        iconBackground: composite.iconBackground,
        metadata: composite.metadata || [],
        latestEvents,
        nextInQueueEvents,
        moreInQueueCount: 0,
      };
    }

    // Handle approval nodes
    if (data.type === "approval" && data.approval) {
      const approval = data.approval;
      const latestEvents: SidebarEvent[] = [];

      if (approval.awaitingEvent) {
        latestEvents.push({
          title: approval.awaitingEvent.title,
          subtitle: approval.awaitingEvent.subtitle,
          state: "waiting",
          isOpen: false,
          receivedAt: approval.receivedAt,
        });
      }

      return {
        title: approval.title,
        iconSlug: approval.iconSlug,
        iconColor: approval.iconColor,
        iconBackground: approval.iconBackground,
        metadata: [],
        latestEvents,
        nextInQueueEvents: [],
        moreInQueueCount: 0,
      };
    }

    // Handle switch nodes
    if (data.type === "switch" && data.switch) {
      const switchData = data.switch;
      const latestEvents: SidebarEvent[] = [];

      // Add the most recent events from each path
      if (switchData.stages) {
        const sortedStages = [...switchData.stages].sort((a, b) => {
          const aTime = a.receivedAt?.getTime() || 0;
          const bTime = b.receivedAt?.getTime() || 0;
          return bTime - aTime;
        });

        sortedStages.slice(0, 5).forEach((stage: any) => {
          if (stage.eventTitle) {
            latestEvents.push({
              title: stage.eventTitle,
              subtitle: stage.pathName,
              state: stage.eventState || "processed",
              isOpen: false,
              receivedAt: stage.receivedAt,
            });
          }
        });
      }

      return {
        title: switchData.title,
        iconSlug: "git-branch",
        iconColor: "text-purple-700",
        iconBackground: "bg-purple-100",
        metadata: [],
        latestEvents,
        nextInQueueEvents: [],
        moreInQueueCount: 0,
      };
    }

    return null;
  };
}
