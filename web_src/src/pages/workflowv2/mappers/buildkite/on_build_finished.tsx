import { useState } from "react";
import { getColorClass, getBackgroundColorClass } from "@/utils/colors";
import { CustomFieldRenderer, NodeInfo, TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import { TriggerProps } from "@/ui/trigger";
import BuildkiteLogo from "@/assets/buildkite-logo.svg";
import { formatTimeAgo } from "@/utils/date";
import { Button } from "@/components/ui/button";
import { Copy, Check, ExternalLink } from "lucide-react";
import { showErrorToast } from "@/utils/toast";

interface OnBuildFinishedMetadata {
  organization?: string;
  pipeline?: string;
  branch?: string;
  appSubscriptionID?: string;
  webhookUrl?: string;
  webhookToken?: string;
  orgSlug?: string;
}

interface OnBuildFinishedEventData {
  build?: {
    id: string;
    state: string;
    result?: string;
    web_url?: string;
    number?: number;
    commit?: string;
    branch?: string;
    message?: string;
    blocked?: boolean;
    started_at?: string;
    finished_at?: string;
  };
  pipeline?: {
    id: string;
    slug: string;
    name: string;
  };
  organization?: {
    id: string;
    slug: string;
    name: string;
  };
  sender?: {
    id: string;
    name: string;
    email: string;
  };
}

/**
 * Renderer for the "buildkite.onBuildFinished" trigger type
 */
export const onBuildFinishedTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const eventData = context.event?.data as OnBuildFinishedEventData;
    const build = eventData?.build;
    const state = build?.state || "";
    const result = build?.blocked ? "blocked" : state;
    const timeAgo = context.event?.createdAt ? formatTimeAgo(new Date(context.event?.createdAt)) : "";
    const subtitle = result && timeAgo ? `${result} · ${timeAgo}` : result || timeAgo;

    return {
      title: eventData?.pipeline?.name || eventData?.build?.web_url?.split("/").pop() || "Unknown Pipeline",
      subtitle,
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as OnBuildFinishedEventData;
    const build = eventData?.build;
    const pipeline = eventData?.pipeline;
    const sender = eventData?.sender;

    const startedAt = build?.started_at ? new Date(build.started_at).toLocaleString() : "";
    const finishedAt = build?.finished_at ? new Date(build.finished_at).toLocaleString() : "";
    const buildUrl = build?.web_url || "";

    return {
      "Started At": startedAt,
      "Finished At": finishedAt,
      "Build State": build?.state || "",
      Pipeline: pipeline?.name || "",
      "Pipeline URL": buildUrl,
      Branch: build?.branch || "",
      Commit: build?.commit || "",
      Message: build?.message || "",
      "Triggered By": sender?.name || "",
    };
  },

  getTriggerProps: (context: TriggerRendererContext) => {
    const { node, definition, lastEvent } = context;
    const metadata = node.metadata as unknown as OnBuildFinishedMetadata;
    const metadataItems = [];

    if (metadata?.pipeline) {
      metadataItems.push({
        icon: "layers",
        label: metadata.pipeline,
      });
    }

    if (metadata?.branch) {
      metadataItems.push({
        icon: "git-branch",
        label: metadata.branch,
      });
    }

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: BuildkiteLogo,
      iconColor: getColorClass(definition.color),
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: metadataItems,
    };

    if (lastEvent) {
      const eventData = lastEvent.data as OnBuildFinishedEventData;
      const build = eventData?.build;
      const state = build?.state || "";
      const result = build?.blocked ? "blocked" : state;
      const timeAgo = lastEvent.createdAt ? formatTimeAgo(new Date(lastEvent.createdAt)) : "";
      const subtitle = result && timeAgo ? `${result} · ${timeAgo}` : result || timeAgo;

      props.lastEventData = {
        title: eventData?.pipeline?.name || "Unknown Pipeline",
        subtitle,
        receivedAt: new Date(lastEvent.createdAt),
        state: "triggered",
        eventId: lastEvent.id,
      };
    }

    return props;
  },
};

export const onBuildFinishedCustomFieldRenderer: CustomFieldRenderer = {
  render: (node: NodeInfo) => {
    const metadata = node.metadata as OnBuildFinishedMetadata | undefined;
    const webhookUrl = metadata?.webhookUrl || " ";
    const webhookToken = metadata?.webhookToken || " ";
    const orgSlug = metadata?.orgSlug;

    const disabled = !orgSlug;

    const handleOpenBuildkite = () => {
      if (orgSlug) {
        window.open(`https://buildkite.com/organizations/${orgSlug}/services/webhook/new`, "_blank");
      }
    };

    const CopyButton: React.FC<{ code: string; disabled?: boolean }> = ({ code, disabled }) => {
      const [copied, setCopied] = useState(false);

      const handleCopy = async () => {
        if (disabled) return;
        try {
          await navigator.clipboard.writeText(code);
          setCopied(true);
          setTimeout(() => setCopied(false), 2000);
        } catch (_err) {
          showErrorToast("Failed to copy text");
        }
      };

      return (
        <Button
          variant="ghost"
          size="sm"
          onClick={handleCopy}
          disabled={disabled}
          className="h-auto p-0 px-1 text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-200 disabled:opacity-50 disabled:cursor-not-allowed"
          title={copied ? "Copied!" : "Copy to clipboard"}
        >
          {copied ? <Check className="w-3 h-3" /> : <Copy className="w-3 h-3" />}
        </Button>
      );
    };

    const FieldLabel = ({ children, disabled }: { children: React.ReactNode; disabled?: boolean }) => (
      <span
        className={`text-xs font-medium ${!disabled ? "text-gray-700 dark:text-gray-200" : "text-gray-500 dark:text-gray-400"}`}
      >
        {children}
      </span>
    );

    const FieldValue = ({ children, disabled }: { children: React.ReactNode; disabled?: boolean }) => (
      <pre
        className={`mt-1 text-xs border-1 px-2.5 py-2 rounded-md font-mono whitespace-pre-wrap break-all ${
          !disabled
            ? "text-gray-800 dark:text-gray-100 border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-900"
            : "text-gray-400 dark:text-gray-500 border-gray-300 dark:border-gray-600 bg-gray-100 dark:bg-gray-800"
        }`}
      >
        {children}
      </pre>
    );

    return (
      <div className="border-t-1 border-gray-200 pt-4">
        <div className="space-y-3">
          <div>
            <span className="text-sm font-medium text-gray-700 dark:text-gray-300">Buildkite Webhook Setup</span>
            <div
              className={`text-xs mt-2 border-1 px-2.5 py-2 rounded-md text-gray-800 dark:text-gray-100 border-yellow-200 dark:border-yellow-700 bg-yellow-50 dark:bg-yellow-900/20`}
            >
              <ol className="list-decimal ml-4 space-y-1">
                <li>
                  <strong>Save the trigger</strong> to generate the webhook URL and token.
                </li>
                <li>Click the button below to create Buildkite webhook.</li>
                <li>Enter provided webhook URL and token.</li>
                <li>Select &quot;build.finished&quot; as the event and choose your pipeline.</li>
              </ol>
              <div className="mt-3">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={handleOpenBuildkite}
                  disabled={disabled}
                  className={disabled ? "opacity-50 cursor-not-allowed" : ""}
                >
                  <ExternalLink className="w-4 h-4" />
                  Create Buildkite webhook
                </Button>
              </div>
              <div className="mt-3">
                <div className="flex items-center gap-1">
                  <FieldLabel disabled={disabled}>Webhook URL</FieldLabel>
                  <CopyButton code={webhookUrl} disabled={disabled} />
                </div>
                <FieldValue disabled={disabled}>{webhookUrl}</FieldValue>
              </div>
              <div className="mt-3">
                <div className="flex items-center gap-1">
                  <FieldLabel disabled={disabled}>Webhook Token</FieldLabel>
                  <CopyButton code={webhookToken} disabled={disabled} />
                </div>
                <FieldValue disabled={disabled}>{webhookToken}</FieldValue>
              </div>
            </div>
          </div>
        </div>
      </div>
    );
  },
};
