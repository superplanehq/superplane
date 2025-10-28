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

      // Add 3 random latest events
      latestEvents.push(
        {
          title: trigger.lastEventData.title + " 1",
          subtitle: trigger.lastEventData.subtitle,
          state: "processed",
          isOpen: false,
          receivedAt: new Date(Date.now() - 5 * 60 * 1000),
          values: { email: "john@example.com", userId: "u_123" },
        },
        {
          title: trigger.lastEventData.title + " 2",
          subtitle: trigger.lastEventData.subtitle,
          state: "processed",
          isOpen: false,
          receivedAt: new Date(Date.now() - 15 * 60 * 1000),
          values: { amount: "49.99", currency: "USD" },
        },
        {
          title: trigger.lastEventData.title + " 3",
          subtitle: trigger.lastEventData.subtitle,
          state: "discarded",
          isOpen: false,
          receivedAt: new Date(Date.now() - 30 * 60 * 1000),
        }
      );

      // Add 2 queue events
      const nextInQueueEvents: SidebarEvent[] = [
        {
          title: "Email verification",
          subtitle: "pending",
          state: "waiting",
          isOpen: false,
          receivedAt: new Date(Date.now() + 5 * 60 * 1000),
        },
        {
          title: "Data sync",
          subtitle: "scheduled",
          state: "waiting",
          isOpen: false,
          receivedAt: new Date(Date.now() + 10 * 60 * 1000),
        },
      ];

      return {
        title: trigger.title,
        iconSrc: trigger.iconSrc,
        iconSlug: trigger.iconSlug,
        iconColor: trigger.iconColor,
        iconBackground: trigger.iconBackground,
        metadata: trigger.metadata || [],
        latestEvents,
        nextInQueueEvents,
        moreInQueueCount: 0,
        hideQueueEvents: true,
      };
    }

    // Handle composite nodes
    if (data.type === "composite" && data.composite) {
      const composite = data.composite;
      const latestEvents: SidebarEvent[] = [];
      const nextInQueueEvents: SidebarEvent[] = [];

      const conversionStateMap: Record<string, string> = {
        success: "processed",
        failed: "discarded",
        running: "processed",
      };

      if (composite.lastRunItem) {
        latestEvents.push({
          title: composite.lastRunItem.title,
          subtitle: composite.lastRunItem.subtitle,
          state: (conversionStateMap[composite.lastRunItem.state] || "processed") as "waiting" | "processed" | "discarded",
          isOpen: false,
          receivedAt: composite.lastRunItem.receivedAt,
          values: composite.lastRunItem.values,
          childEventsInfo: composite.lastRunItem.childEventsInfo,
        });
      }

      // Add 3 random latest events
      latestEvents.push(
        {
          title: composite.lastRunItem.title + " 1",
          subtitle: composite.lastRunItem.subtitle,
          state: "processed",
          isOpen: false,
          receivedAt: new Date(Date.now() - 2 * 60 * 1000),
          values: { duration: "1.2s", steps: "5" },
        },
        {
          title: composite.lastRunItem.title + " 2",
          subtitle: composite.lastRunItem.subtitle,
          state: "processed",
          isOpen: false,
          receivedAt: new Date(Date.now() - 8 * 60 * 1000),
          values: { records: "1,250", format: "JSON" },
          childEventsInfo: {
           count: 1,
           waitingInfos: [
            {
              icon: "Calendar",
              info: "20 minutes to transform"
            },
           ] 
          }
        },
        {
          title: composite.lastRunItem.title + " 3",
          subtitle: composite.lastRunItem.subtitle,
          state: "discarded",
          isOpen: false,
          receivedAt: new Date(Date.now() - 20 * 60 * 1000),
        }
      );

      if (composite.nextInQueue) {
        nextInQueueEvents.push({
          title: composite.nextInQueue.title,
          subtitle: composite.nextInQueue.subtitle,
          state: "waiting",
          isOpen: false,
          receivedAt: composite.nextInQueue.receivedAt,
        });
      }

      // Add 2 additional queue events
      nextInQueueEvents.push(
        {
          title: composite.lastRunItem.title + " 1",
          subtitle: composite.lastRunItem.subtitle,
          state: "waiting",
          isOpen: false,
          receivedAt: new Date(Date.now() + 15 * 60 * 1000),
        },
        {
          title: composite.lastRunItem.title + " 2",
          subtitle: composite.lastRunItem.subtitle,
          state: "waiting",
          isOpen: false,
          receivedAt: new Date(Date.now() + 25 * 60 * 1000),
        }
      );

      return {
        title: composite.title,
        iconSrc: composite.iconSrc,
        iconSlug: composite.iconSlug,
        iconColor: composite.iconColor,
        iconBackground: composite.iconBackground,
        metadata: composite.metadata || [],
        latestEvents,
        nextInQueueEvents,
        moreInQueueCount: nextInQueueEvents.length > 2 ? nextInQueueEvents.length - 2 : 0,
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

      // Add 3 random latest events
      latestEvents.push(
        {
          title: "FEAT: Add new feature",
          subtitle: "PR #142 approved",
          state: "waiting",
          isOpen: false,
          receivedAt: new Date(Date.now() - 1 * 60 * 1000),
          values: { amount: "2500", department: "Marketing" },
        },
        {
          title: "FIX: Fix bug",
          subtitle: "PR #142 approved",
          state: "processed",
          isOpen: false,
          receivedAt: new Date(Date.now() - 10 * 60 * 1000),
          values: { pullRequest: "142", reviewer: "alice" },
        },
        {
          title: "REF: Fix bug",
          subtitle: "denied",
          state: "discarded",
          isOpen: false,
          receivedAt: new Date(Date.now() - 35 * 60 * 1000),
        }
      );

      // Add 2 queue events
      const nextInQueueEvents: SidebarEvent[] = [
        {
          title: "FEAT: Add new feature",
          subtitle: "pending legal review",
          state: "waiting",
          isOpen: false,
          receivedAt: new Date(Date.now() + 3 * 60 * 1000),
        },
        {
          title: "FIX: Fix bug",
          subtitle: "pending legal review",
          state: "waiting",
          isOpen: false,
          receivedAt: new Date(Date.now() + 8 * 60 * 1000),
        },
      ];

      return {
        title: approval.title,
        iconSlug: approval.iconSlug,
        iconColor: approval.iconColor,
        iconBackground: approval.iconBackground,
        metadata: [],
        latestEvents,
        nextInQueueEvents,
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

      // Add 3 additional random events
      latestEvents.push(
        {
          title: "Route A executed",
          subtitle: "premium_users",
          state: "processed",
          isOpen: false,
          receivedAt: new Date(Date.now() - 3 * 60 * 1000),
          values: { path: "premium", users: "45" },
        },
        {
          title: "Route B executed",
          subtitle: "basic_users",
          state: "processed",
          isOpen: false,
          receivedAt: new Date(Date.now() - 12 * 60 * 1000),
          values: { path: "basic", users: "127" },
        },
        {
          title: "Route validation failed",
          subtitle: "unknown_path",
          state: "discarded",
          isOpen: false,
          receivedAt: new Date(Date.now() - 25 * 60 * 1000),
        }
      );

      // Add 2 queue events
      const nextInQueueEvents: SidebarEvent[] = [
        {
          title: "Batch route processing",
          subtitle: "enterprise_users",
          state: "waiting",
          isOpen: false,
          receivedAt: new Date(Date.now() + 7 * 60 * 1000),
        },
        {
          title: "Analytics routing",
          subtitle: "data_pipeline",
          state: "waiting",
          isOpen: false,
          receivedAt: new Date(Date.now() + 12 * 60 * 1000),
        },
      ];

      return {
        title: switchData.title,
        iconSlug: "git-branch",
        iconColor: "text-purple-700",
        iconBackground: "bg-purple-100",
        metadata: [],
        latestEvents,
        nextInQueueEvents,
        moreInQueueCount: 0,
      };
    }

    return null;
  };
}
