import { CustomFieldRenderer, NodeInfo } from "../types";
import React from "react";

type OnAlertFiredMetadata = {
  webhookUrl?: string;
  sharedSecret?: string;
};

function extractWebhookData(node: NodeInfo): { webhookUrl: string; sharedSecret: string } {
  const md = (node.metadata ?? {}) as OnAlertFiredMetadata;

  const webhookUrl = (md.webhookUrl ?? "").trim();
  const sharedSecret = (md.sharedSecret ?? "").trim();

  return { webhookUrl, sharedSecret };
}

function CopyRow({ label, value }: { label: string; value: string }): React.ReactElement {
  const shown = value?.trim() ? value : "Save this node to generate";

  const [copied, setCopied] = React.useState(false);

  const onCopy = async () => {
    if (!value?.trim()) return;
    try {
      await navigator.clipboard.writeText(value);
      setCopied(true);
      setTimeout(() => setCopied(false), 1200);
    } catch {
      // ignore
    }
  };

  return (
    <div style={{ display: "flex", gap: 8, alignItems: "center", marginTop: 10 }}>
      <div style={{ flex: 1 }}>
        <div style={{ fontSize: 12, opacity: 0.7 }}>{label}</div>
        <div style={{ fontFamily: "monospace", fontSize: 12, wordBreak: "break-all" }}>{shown}</div>
      </div>

      <button
        type="button"
        onClick={onCopy}
        disabled={!value?.trim()}
        style={{
          padding: "4px 8px",
          borderRadius: 8,
          border: "1px solid rgba(0,0,0,0.15)",
          cursor: value?.trim() ? "pointer" : "not-allowed",
          opacity: value?.trim() ? 1 : 0.5,
          background: copied ? "#d1fae5" : "transparent",
          transition: "background 150ms ease",
          fontWeight: 600,
          minWidth: 72,
        }}
      >
        {copied ? "Copied" : "Copy"}
      </button>
    </div>
  );
}

export const honeycombOnAlertFiredCustomFieldRenderer: CustomFieldRenderer = {
  render(node: NodeInfo) {
    const { webhookUrl, sharedSecret } = extractWebhookData(node);

    return (
      <div style={{ marginTop: 12 }}>
        <div style={{ fontWeight: 600, marginBottom: 6 }}>Honeycomb webhook setup</div>
        <div style={{ fontSize: 12, opacity: 0.8 }}>
          After saving this trigger, SuperPlane will generate a shared Webhook URL and Shared Secret.
        </div>

        <CopyRow label="Webhook URL" value={webhookUrl} />
        <CopyRow label="Shared Secret" value={sharedSecret} />

        <div style={{ marginTop: 10, fontSize: 12, opacity: 0.7 }}>
          If no events arrive, verify the webhook is configured as a recipient in Honeycomb.
        </div>
      </div>
    );
  },
};

export const honeycombCustomFieldRenderers: Record<string, CustomFieldRenderer> = {
  onAlertFired: honeycombOnAlertFiredCustomFieldRenderer,
  "honeycomb.onAlertFired": honeycombOnAlertFiredCustomFieldRenderer,
};
