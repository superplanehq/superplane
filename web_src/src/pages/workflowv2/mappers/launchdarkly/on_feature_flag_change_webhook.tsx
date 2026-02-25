import { CustomFieldRenderer, NodeInfo } from "../types";

interface OnFeatureFlagChangeMetadata {
  webhookUrl?: string;
}

export const onFeatureFlagChangeCustomFieldRenderer: CustomFieldRenderer = {
  render: (node: NodeInfo) => {
    const metadata = node.metadata as OnFeatureFlagChangeMetadata | undefined;

    if (!metadata?.webhookUrl) {
      return (
        <div className="border-t-1 border-gray-200 dark:border-gray-700 pt-4">
          <div className="rounded-md border border-amber-200 dark:border-amber-800 bg-amber-50 dark:bg-amber-950/40 px-3 py-2.5">
            <p className="text-xs font-medium text-amber-800 dark:text-amber-200 mb-1">
              Webhook not yet created
            </p>
            <p className="text-xs text-amber-700 dark:text-amber-300">
              Save the canvas to automatically create the webhook in LaunchDarkly. No manual setup is
              required.
            </p>
          </div>
        </div>
      );
    }

    return (
      <div className="border-t-1 border-gray-200 dark:border-gray-700 pt-4">
        <div className="text-xs text-gray-600 dark:text-gray-400 rounded-md border border-gray-200 dark:border-gray-600 px-3 py-2.5 bg-gray-50 dark:bg-gray-800/50">
          <p className="font-medium text-gray-700 dark:text-gray-300 mb-1">
            LaunchDarkly webhook is being configured automatically
          </p>
          <p>
            SuperPlane creates and manages the webhook in LaunchDarkly using your API access token.
            Events will be received once the webhook is provisioned.
          </p>
        </div>
      </div>
    );
  },
};
