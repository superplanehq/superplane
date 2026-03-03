import { getBackgroundColorClass } from "@/utils/colors";
import { CustomFieldRenderer, NodeInfo, TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import { TriggerProps } from "@/ui/trigger";
import { formatTimeAgo } from "@/utils/date";
import { MetadataItem } from "@/ui/metadataList";
import { OnPackageEventMetadata } from "./types";
import { stringOrDash } from "../utils";
import cloudsmithIcon from "@/assets/icons/integrations/cloudsmith.svg";

interface OnPackageEventConfiguration {
  repository?: string;
  events?: string[];
}

interface PackageEventData {
  event?: string;
  package?: {
    name?: string;
    version?: string;
    format?: string;
    namespace?: string;
    repository?: string;
  };
}

/**
 * Renderer for the "cloudsmith.onPackageEvent" trigger
 */
export const onPackageEventTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const eventData = context.event?.data as PackageEventData;
    const pkgName = eventData?.package?.name;
    const version = eventData?.package?.version;
    const eventType = eventData?.event;

    const title = pkgName ? `${pkgName}${version ? `@${version}` : ""}` : eventType || "Package event";
    const subtitle = context.event?.createdAt ? formatTimeAgo(new Date(context.event.createdAt)) : "";

    return { title, subtitle };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as PackageEventData;
    const pkg = eventData?.package;

    return {
      Event: stringOrDash(eventData?.event),
      Package: stringOrDash(pkg?.name),
      Version: stringOrDash(pkg?.version),
      Format: stringOrDash(pkg?.format),
      Repository: pkg?.namespace ? `${pkg.namespace}/${pkg.repository}` : stringOrDash(pkg?.repository),
    };
  },

  getTriggerProps: (context: TriggerRendererContext) => {
    const { node, definition, lastEvent } = context;
    const metadata = node.metadata as OnPackageEventMetadata | undefined;
    const configuration = node.configuration as OnPackageEventConfiguration | undefined;
    const metadataItems: MetadataItem[] = [];

    if (metadata?.repository) {
      metadataItems.push({
        icon: "package",
        label: metadata.repository,
      });
    }

    if (configuration?.events?.length) {
      metadataItems.push({
        icon: "zap",
        label: configuration.events.join(", "),
      });
    }

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: cloudsmithIcon,
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: metadataItems,
    };

    if (lastEvent) {
      const { title, subtitle } = onPackageEventTriggerRenderer.getTitleAndSubtitle({ event: lastEvent });
      props.lastEventData = {
        title,
        subtitle,
        receivedAt: new Date(lastEvent.createdAt!),
        state: "triggered",
        eventId: lastEvent.id!,
      };
    }

    return props;
  },
};

export const onPackageEventCustomFieldRenderer: CustomFieldRenderer = {
  render: (node: NodeInfo) => {
    const metadata = node.metadata as OnPackageEventMetadata | undefined;
    const repository = metadata?.repository || "[REPOSITORY]";
    const webhookUrl = metadata?.webhookUrl || "[URL GENERATED ONCE THE CANVAS IS SAVED]";

    return (
      <div className="border-t-1 border-gray-200 pt-4">
        <div className="space-y-3">
          <div>
            <span className="text-sm font-medium text-gray-700 dark:text-gray-300">Cloudsmith Webhook</span>
            <div className="text-xs text-gray-800 dark:text-gray-100 mt-2 border-1 border-gray-300 dark:border-gray-600 px-2.5 py-2 bg-gray-50 dark:bg-gray-800 rounded-md">
              <p>
                A webhook for <strong>{repository}</strong> will be created in Cloudsmith automatically when the canvas
                is saved.
              </p>
              <div className="mt-3">
                <span className="text-xs font-medium text-gray-700 dark:text-gray-200">Webhook URL</span>
                <div className="relative group mt-1">
                  <pre className="text-xs text-gray-800 dark:text-gray-100 border-1 border-gray-300 dark:border-gray-600 px-2.5 py-2 bg-white dark:bg-gray-900 rounded-md font-mono whitespace-pre-wrap break-all">
                    {webhookUrl}
                  </pre>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    );
  },
};
