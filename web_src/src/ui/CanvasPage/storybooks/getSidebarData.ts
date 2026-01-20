import { CanvasNode, SidebarData } from "../index";
import { genCommit } from "./commits";
import { DockerImage, genDockerImage } from "./dockerImages";
import { SidebarEvent } from "@/ui/componentSidebar/types";

const isGitSha = (subtitle: string) => (subtitle?.length || 0) === 8;

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
      const latestEvents: SidebarEvent[] = data.trigger.latestEvents || [];

      if (trigger.lastEventData) {
        latestEvents.push({
          id: trigger.lastEventData.id,
          title: trigger.lastEventData.title,
          subtitle: trigger.lastEventData.subtitle,
          state: trigger.lastEventData.state || "processed",
          isOpen: false,
          receivedAt: trigger.lastEventData.receivedAt,
          values: trigger.lastEventData.values,
        });
      }

      const genFunction: () => DockerImage = isGitSha(trigger.lastEventData.subtitle)
        ? (genCommit as () => DockerImage)
        : genDockerImage;

      // Add 3 random latest events
      latestEvents.push(
        {
          id: "550e8400-e29b-41d4-a716-446655440000",
          title: genFunction().message,
          subtitle: genFunction()?.size || genFunction()?.sha,
          state: "triggered",
          isOpen: false,
          receivedAt: new Date(Date.now() - 5 * 60 * 1000),
          values: { email: "john@example.com", userId: "u_123" },
        },
        {
          id: "550e8400-e29b-41d4-a716-446655440001",
          title: genFunction().message,
          subtitle: genFunction()?.size || genFunction()?.sha,
          state: "triggered",
          isOpen: false,
          receivedAt: new Date(Date.now() - 15 * 60 * 1000),
          values: { amount: "49.99", currency: "USD" },
        },
        {
          id: "550e8400-e29b-41d4-a716-446655440002",
          title: genFunction().message,
          subtitle: genFunction()?.size || genFunction()?.sha,
          state: "discarded",
          isOpen: false,
          receivedAt: new Date(Date.now() - 30 * 60 * 1000),
        },
      );

      // Add 2 queue events
      const nextInQueueEvents: SidebarEvent[] = [
        {
          id: "650e8400-e29b-41d4-a716-446655440002",
          title: genFunction().message,
          subtitle: genFunction()?.size || genFunction()?.sha,
          state: "waiting",
          isOpen: false,
          receivedAt: new Date(Date.now() + 5 * 60 * 1000),
        },
        {
          id: "750e8400-e29b-41d4-a716-446655440002",
          title: genFunction().message,
          subtitle: genFunction()?.size || genFunction()?.sha,
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
        latestEvents,
        nextInQueueEvents,
        totalInQueueCount: 0,
        totalInHistoryCount: 0,
        hideQueueEvents: true,
      };
    }

    // Handle composite nodes
    if (data.type === "composite" && data.composite) {
      const composite = data.composite;
      const latestEvents: SidebarEvent[] = [];
      const nextInQueueEvents: SidebarEvent[] = [];

      const genFunction: () => DockerImage = isGitSha(composite?.lastRunItem?.subtitle)
        ? (genCommit as () => DockerImage)
        : genDockerImage;

      const conversionStateMap: Record<string, string> = {
        success: "processed",
        fail: "discarded",
        running: "running",
      };

      if (composite?.lastRunItem) {
        latestEvents.push({
          id: composite.lastRunItem.id,
          title: composite.lastRunItem.title,
          subtitle: composite.lastRunItem.subtitle,
          state: (conversionStateMap[composite.lastRunItem.state] || "processed") as
            | "waiting"
            | "processed"
            | "discarded",
          isOpen: false,
          receivedAt: composite.lastRunItem.receivedAt,
          values: composite.lastRunItem.values,
        });
      }

      // Add 3 random latest events
      latestEvents.push(
        {
          id: "850e8400-e29b-41d4-a716-446655440002",
          title: genFunction().message,
          subtitle: genFunction()?.size || genFunction()?.sha,
          state: "triggered",
          isOpen: false,
          receivedAt: new Date(Date.now() - 2 * 60 * 1000),
          values: { duration: "1.2s", steps: "5" },
        },
        {
          id: "950e8400-e29b-41d4-a716-446655440001",
          title: genFunction().message,
          subtitle: genFunction()?.size || genFunction()?.sha,
          state: "triggered",
          isOpen: false,
          receivedAt: new Date(Date.now() - 8 * 60 * 1000),
          values: { records: "1,250", format: "JSON" },
        },
        {
          id: "1050e8400-e29b-41d4-a716-446655440002",
          title: genFunction().message,
          subtitle: genFunction()?.size || genFunction()?.sha,
          state: "discarded",
          isOpen: false,
          receivedAt: new Date(Date.now() - 20 * 60 * 1000),
        },
      );

      if (composite.nextInQueue) {
        nextInQueueEvents.push({
          id: "650e8400-e29b-41d4-a716-446655440002",
          title: genFunction().message,
          subtitle: genFunction()?.size || genFunction()?.sha,
          state: "waiting",
          isOpen: false,
          receivedAt: composite.nextInQueue.receivedAt,
        });
      }

      // Add 2 additional queue events
      nextInQueueEvents.push(
        {
          id: "950e8400-e29b-41d4-a716-446655440002",
          title: genFunction().message,
          subtitle: genFunction()?.size || genFunction()?.sha,
          state: "waiting",
          isOpen: false,
          receivedAt: new Date(Date.now() + 15 * 60 * 1000),
        },
        {
          id: "1050e8400-e29b-41d4-a716-446655440002",
          title: genFunction().message,
          subtitle: genFunction()?.size || genFunction()?.sha,
          state: "waiting",
          isOpen: false,
          receivedAt: new Date(Date.now() + 25 * 60 * 1000),
        },
      );

      return {
        title: composite.title,
        iconSrc: composite.iconSrc,
        iconSlug: composite.iconSlug,
        iconColor: composite.iconColor,
        latestEvents,
        nextInQueueEvents,
        totalInQueueCount: nextInQueueEvents.length > 2 ? nextInQueueEvents.length - 2 : 0,
        totalInHistoryCount: latestEvents.length > 2 ? latestEvents.length - 2 : 0,
      };
    }

    // Handle approval nodes
    if (data.type === "approval" && data.approval) {
      const approval = data.approval;
      const latestEvents: SidebarEvent[] = [];

      const genFunction: () => DockerImage = isGitSha(approval?.awaitingEvent?.subtitle)
        ? (genCommit as () => DockerImage)
        : genDockerImage;

      if (approval?.awaitingEvent) {
        latestEvents.push({
          id: approval.awaitingEvent.id,
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
          id: "250e8400-e29b-41d4-a716-446655440000",
          title: genFunction().message,
          subtitle: genFunction()?.size || genFunction()?.sha,
          state: "triggered",
          isOpen: false,
          receivedAt: new Date(Date.now() - 1 * 60 * 1000),
          values: { amount: "2500", department: "Marketing" },
        },
        {
          id: "350e8400-e29b-41d4-a716-446655440001",
          title: genFunction().message,
          subtitle: genFunction()?.size || genFunction()?.sha,
          state: "triggered",
          isOpen: false,
          receivedAt: new Date(Date.now() - 10 * 60 * 1000),
          values: { pullRequest: "142", reviewer: "alice" },
        },
        {
          id: "450e8400-e29b-41d4-a716-446655440002",
          title: genFunction().message,
          subtitle: genFunction()?.size || genFunction()?.sha,
          state: "discarded",
          isOpen: false,
          receivedAt: new Date(Date.now() - 35 * 60 * 1000),
        },
      );

      // Add 2 queue events
      const nextInQueueEvents: SidebarEvent[] = [
        {
          id: "550e8400-e29b-41d4-a716-446655440002",
          title: genFunction().message,
          subtitle: genFunction()?.size || genFunction()?.sha,
          state: "waiting",
          isOpen: false,
          receivedAt: new Date(Date.now() + 3 * 60 * 1000),
        },
        {
          id: "650e8400-e29b-41d4-a716-446655440002",
          title: genFunction().message,
          subtitle: genFunction()?.size || genFunction()?.sha,
          state: "waiting",
          isOpen: false,
          receivedAt: new Date(Date.now() + 8 * 60 * 1000),
        },
      ];

      return {
        title: approval.title,
        iconSlug: approval.iconSlug,
        iconColor: approval.iconColor,
        latestEvents,
        nextInQueueEvents,
        totalInQueueCount: 0,
        totalInHistoryCount: 0,
      };
    }

    // Handle switch nodes
    if (data.type === "switch" && data.switch) {
      const genFunction: () => DockerImage = isGitSha(data.switch.subtitle)
        ? (genCommit as () => DockerImage)
        : genDockerImage;
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
              id: stage.id,
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
          id: "750e8400-e29b-41d4-a716-446655440000",
          title: genFunction().message,
          subtitle: genFunction()?.size || genFunction()?.sha,
          state: "triggered",
          isOpen: false,
          receivedAt: new Date(Date.now() - 3 * 60 * 1000),
          values: { path: "premium", users: "45" },
        },
        {
          id: "850e8400-e29b-41d4-a716-446655440001",
          title: genFunction().message,
          state: "triggered",
          isOpen: false,
          receivedAt: new Date(Date.now() - 12 * 60 * 1000),
          values: { path: "basic", users: "127" },
        },
        {
          id: "950e8400-e29b-41d4-a716-446655440002",
          title: genFunction().message,
          subtitle: genFunction()?.size || genFunction()?.sha,
          state: "discarded",
          isOpen: false,
          receivedAt: new Date(Date.now() - 25 * 60 * 1000),
        },
      );

      // Add 2 queue events
      const nextInQueueEvents: SidebarEvent[] = [
        {
          id: "850e8400-e29b-41d4-a716-446655440002",
          title: genFunction().message,
          subtitle: genFunction()?.size || genFunction()?.sha,
          state: "waiting",
          isOpen: false,
          receivedAt: new Date(Date.now() + 7 * 60 * 1000),
        },
        {
          id: "950e8400-e29b-41d4-a716-446655440002",
          title: genFunction().message,
          subtitle: genFunction()?.size || genFunction()?.sha,
          state: "waiting",
          isOpen: false,
          receivedAt: new Date(Date.now() + 12 * 60 * 1000),
        },
      ];

      return {
        title: switchData.title,
        iconSlug: "git-branch",
        iconColor: "text-purple-700",
        latestEvents,
        nextInQueueEvents,
        totalInQueueCount: 0,
        totalInHistoryCount: 0,
      };
    }

    return null;
  };
}
