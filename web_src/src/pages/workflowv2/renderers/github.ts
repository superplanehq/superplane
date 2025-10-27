import { ComponentsNode, TriggersTrigger } from "@/api-client";
import { getColorClass, getBackgroundColorClass } from "@/utils/colors";
import { TriggerRenderer } from "./types";
import githubIcon from '@/assets/icons/integrations/github.svg';

interface GitHubMetadata {
  repository: {
    id: string;
    name: string;
    url: string;
  }
}

/**
 * Renderer for the "github" trigger type
 */
export const githubTriggerRenderer: TriggerRenderer = {
  getTriggerProps: (node: ComponentsNode, trigger: TriggersTrigger) => {
    const metadata = node.metadata as unknown as GitHubMetadata;

    return {
      title: node.name!,
      iconSrc: githubIcon,
      iconBackground: "bg-black",
      iconColor: getColorClass(trigger.color),
      headerColor: getBackgroundColorClass(trigger.color),
      collapsedBackground: getBackgroundColorClass(trigger.color),
      metadata: [
        {
          icon: "book",
          label: metadata.repository.name,
        },
      ],
      zeroStateText: "Waiting for the first push...",
    };
  },
};
