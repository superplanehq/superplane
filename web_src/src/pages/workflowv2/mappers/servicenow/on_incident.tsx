import { useState } from "react";
import { getBackgroundColorClass } from "@/utils/colors";
import { CustomFieldRenderer, NodeInfo, TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import { TriggerProps } from "@/ui/trigger";
import { Icon } from "@/components/Icon";
import { canvasesInvokeNodeTriggerAction } from "@/api-client";
import { useQueryClient } from "@tanstack/react-query";
import { useParams } from "react-router-dom";
import { withOrganizationHeader } from "@/utils/withOrganizationHeader";
import { canvasKeys } from "@/hooks/useCanvasData";
import { showErrorToast } from "@/utils/toast";
import snIcon from "@/assets/icons/integrations/servicenow.svg";
import { OnIncidentConfiguration, ServiceNowIncident, STATE_LABELS, URGENCY_LABELS, IMPACT_LABELS } from "./types";
import { buildSubtitle } from "../utils";

interface OnIncidentMetadata {
  webhookUrl?: string;
}

interface OnIncidentEventData extends ServiceNowIncident {}

export const onIncidentTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const incident = context.event?.data as OnIncidentEventData;
    const stateLabel = incident?.state ? STATE_LABELS[incident.state] || incident.state : "";
    const urgencyLabel = incident?.urgency ? URGENCY_LABELS[incident.urgency] || incident.urgency : "";
    const contentParts = [stateLabel, urgencyLabel].filter(Boolean).join(" · ");
    const subtitle = buildSubtitle(contentParts, context.event?.createdAt);

    return {
      title: `${incident?.number || ""} - ${incident?.short_description || ""}`,
      subtitle,
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const incident = context.event?.data as OnIncidentEventData;
    return getDetailsForIncident(incident);
  },

  getTriggerProps: (context: TriggerRendererContext) => {
    const { node, definition, lastEvent } = context;
    const configuration = node.configuration as OnIncidentConfiguration;
    const metadataItems = [];

    if (configuration.events) {
      metadataItems.push({
        icon: "funnel",
        label: `Events: ${configuration.events.join(", ")}`,
      });
    }

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: snIcon,
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: metadataItems,
    };

    if (lastEvent) {
      const incident = lastEvent.data as OnIncidentEventData;
      const stateLabel = incident?.state ? STATE_LABELS[incident.state] || incident.state : "";
      const urgencyLabel = incident?.urgency ? URGENCY_LABELS[incident.urgency] || incident.urgency : "";
      const contentParts = [stateLabel, urgencyLabel].filter(Boolean).join(" · ");
      const subtitle = buildSubtitle(contentParts, lastEvent.createdAt);

      props.lastEventData = {
        title: `${incident?.number || ""} - ${incident?.short_description || ""}`,
        subtitle,
        receivedAt: new Date(lastEvent.createdAt),
        state: "triggered",
        eventId: lastEvent.id,
      };
    }

    return props;
  },
};

function buildBusinessRuleScript(webhookUrl: string, webhookSecret: string): string {
  return `(function executeRule(current, previous) {
    try {
        var eventType = current.operation();

        var body = {
            event_type: eventType,
            incident: {
                sys_id: current.sys_id.toString(),
                number: current.number.toString(),
                short_description: current.short_description.toString(),
                description: current.description.toString(),
                state: current.state.toString(),
                urgency: current.urgency.toString(),
                impact: current.impact.toString(),
                priority: current.priority.toString(),
                category: current.category.toString(),
                assignment_group: { display_value: current.assignment_group.getDisplayValue() },
                assigned_to: { display_value: current.assigned_to.getDisplayValue() },
                caller_id: { display_value: current.caller_id.getDisplayValue() },
                sys_created_on: current.sys_created_on.toString(),
                sys_updated_on: current.sys_updated_on.toString()
            }
        };

        var request = new sn_ws.RESTMessageV2();
        request.setEndpoint('${webhookUrl}');
        request.setHttpMethod('POST');
        request.setRequestHeader('Content-Type', 'application/json');
        request.setRequestHeader('X-Webhook-Secret', '${webhookSecret}');
        request.setRequestBody(JSON.stringify(body));
        request.executeAsync();
    } catch (e) {
        gs.error('Superplane webhook failed: ' + e.message);
    }
})(current, previous);`;
}

function OnIncidentCustomField({ node }: { node: NodeInfo }) {
  const metadata = node.metadata as OnIncidentMetadata | undefined;
  const webhookUrl = metadata?.webhookUrl || "[URL GENERATED ONCE THE CANVAS IS SAVED]";

  const [isResetting, setIsResetting] = useState(false);
  const [secret, setSecret] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);
  const queryClient = useQueryClient();
  const { organizationId, canvasId } = useParams<{ organizationId: string; canvasId: string }>();

  const scriptContent = buildBusinessRuleScript(webhookUrl, secret || "YOUR_SECRET_HERE");

  const handleGenerateSecret = async () => {
    if (!canvasId || !node.id) return;
    setIsResetting(true);
    try {
      const response = await canvasesInvokeNodeTriggerAction(
        withOrganizationHeader({
          path: {
            canvasId: canvasId,
            nodeId: node.id,
            actionName: "resetAuthentication",
          },
          body: { parameters: {} },
        }),
      );
      const newSecret = response.data?.result?.secret as string | undefined;
      if (newSecret) {
        setSecret(newSecret);
        if (organizationId) {
          queryClient.invalidateQueries({
            queryKey: canvasKeys.detail(organizationId, canvasId),
          });
        }
      }
    } catch (_error) {
      showErrorToast("Failed to generate webhook secret");
    } finally {
      setIsResetting(false);
    }
  };

  const handleCopyScript = async () => {
    try {
      await navigator.clipboard.writeText(scriptContent);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch (_error) {
      showErrorToast("Failed to copy script");
    }
  };

  return (
    <div className="border-t-1 border-gray-200 pt-4">
      <div className="space-y-3">
        <div>
          <span className="text-sm font-medium text-gray-700 dark:text-gray-300">ServiceNow Business Rule Setup</span>
          <div className="text-xs text-gray-800 dark:text-gray-100 mt-2 border-1 border-gray-300 dark:border-gray-600 px-2.5 py-2 bg-gray-50 dark:bg-gray-800 rounded-md">
            <ol className="list-decimal ml-4 space-y-1">
              <li>
                In ServiceNow, go to <strong>System Definition &gt; Business Rules</strong> and create a new rule
              </li>
              <li>
                Set the table to <code>incident</code>, set <strong>When</strong> to <strong>after</strong>, and check{" "}
                <strong>insert</strong>, <strong>update</strong>, and/or <strong>delete</strong> as needed
              </li>
              <li>
                Check <strong>Advanced</strong> and paste the script below into the <strong>Script</strong> field
              </li>
            </ol>

            <div className="mt-3 space-y-2">
              {metadata?.webhookUrl ? (
                <>
                  <button
                    onClick={handleGenerateSecret}
                    disabled={isResetting}
                    className="inline-flex items-center gap-2 px-3 py-1.5 text-sm font-medium text-white bg-black hover:bg-gray-700 disabled:bg-gray-400 rounded-md transition-colors"
                  >
                    <Icon
                      name={isResetting ? "loader" : "refresh-ccw"}
                      size="sm"
                      className={isResetting ? "animate-spin" : ""}
                    />
                    {isResetting ? "Generating..." : secret ? "Reset Secret" : "Generate Secret"}
                  </button>
                  {secret && (
                    <div className="p-2 bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-700 rounded-md">
                      <div className="flex items-center gap-2">
                        <Icon name="triangle-alert" size="sm" className="text-yellow-600 dark:text-yellow-400" />
                        <p className="text-xs text-yellow-700 dark:text-yellow-300">
                          Secret: <code className="font-mono font-bold">{secret}</code> — copy it now, it won't be shown
                          again.
                        </p>
                      </div>
                    </div>
                  )}
                </>
              ) : (
                <p className="text-xs text-gray-500 dark:text-gray-400 italic">
                  Save to generate the webhook URL and secret.
                </p>
              )}
            </div>

            <div className="mt-3">
              <div className="flex items-center justify-between">
                <span className="text-xs font-medium text-gray-700 dark:text-gray-200">Business Rule Script</span>
                <button
                  onClick={handleCopyScript}
                  className="inline-flex items-center gap-1 px-2 py-1 text-xs text-gray-600 dark:text-gray-300 hover:text-gray-900 dark:hover:text-white transition-colors"
                >
                  <Icon name={copied ? "check" : "copy"} size="sm" />
                  {copied ? "Copied" : "Copy"}
                </button>
              </div>
              <div className="relative group mt-1">
                <pre className="text-xs text-gray-800 dark:text-gray-100 border-1 border-gray-300 dark:border-gray-600 px-2.5 py-2 bg-white dark:bg-gray-900 rounded-md font-mono whitespace-pre-wrap break-all">
                  {scriptContent}
                </pre>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

export const onIncidentCustomFieldRenderer: CustomFieldRenderer = {
  render: (node: NodeInfo) => {
    return <OnIncidentCustomField node={node} />;
  },
};

function getDetailsForIncident(incident?: OnIncidentEventData): Record<string, string> {
  const details: Record<string, string> = {};

  if (incident?.number) details["Number"] = incident.number;
  if (incident?.short_description) details["Short Description"] = incident.short_description;
  if (incident?.state) details["State"] = STATE_LABELS[incident.state] || incident.state;
  if (incident?.urgency) details["Urgency"] = URGENCY_LABELS[incident.urgency] || incident.urgency;
  if (incident?.impact) details["Impact"] = IMPACT_LABELS[incident.impact] || incident.impact;
  if (incident?.priority) details["Priority"] = incident.priority;
  if (incident?.category) details["Category"] = incident.category;
  if (incident?.assignment_group?.display_value) details["Assignment Group"] = incident.assignment_group.display_value;
  if (incident?.assigned_to?.display_value) details["Assigned To"] = incident.assigned_to.display_value;
  if (incident?.sys_created_on) details["Created On"] = incident.sys_created_on;

  return details;
}
