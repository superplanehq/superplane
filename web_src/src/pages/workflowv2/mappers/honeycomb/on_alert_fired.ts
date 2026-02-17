import { TriggerRenderer, TriggerRendererContext, TriggerEventContext } from "../types";
import { defaultTriggerRenderer } from "../default";

type OnAlertFiredMetadata = {
  webhookUrl?: string;
  sharedSecret?: string;
};

export const onAlertFiredTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext) => {
    const title = context.event?.customName?.trim() || "On Alert Fired";
    return { title, subtitle: "Triggers when Honeycomb sends an alert webhook" };
  },

  getRootEventValues: (context: TriggerEventContext) => ({
    type: context.event?.type,
    createdAt: context.event?.createdAt,
    data: context.event?.data,
  }),

  getTriggerProps: (context: TriggerRendererContext) => {
    const base = defaultTriggerRenderer.getTriggerProps(context);

    const md = (context.node?.metadata || {}) as OnAlertFiredMetadata;
    const webhookUrl = (md.webhookUrl || "").trim();
    const sharedSecret = (md.sharedSecret || "").trim();

    const hasGenerated = !!webhookUrl && !!sharedSecret;

    return {
      ...base,
      nodeMeta: {
        title: "Honeycomb webhook setup",
        sections: [
          {
            title: "Copy into Honeycomb",
            items: [
              {
                label: "Webhook URL",
                value: hasGenerated ? webhookUrl : "Save this trigger to generate",
                copyable: hasGenerated,
              },
              {
                label: "Shared Secret",
                value: hasGenerated ? sharedSecret : "Save this trigger to generate",
                copyable: hasGenerated,
              },
            ],
          },
          {
            title: "Honeycomb steps",
            items: [
              {
                label: "Where to paste",
                value:
                  "Honeycomb → Team Settings → Integrations → Webhooks. Create a webhook integration and paste the values above. Then attach the webhook as an alert recipient.",
              },
            ],
          },
          {
            title: "Troubleshooting",
            items: [
              {
                label: "No events?",
                value:
                  "If no events arrive, verify the webhook is configured as a recipient in Honeycomb and trigger an alert to test.",
              },
            ],
          },
        ],
      },
    };
  },
};
