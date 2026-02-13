import { CustomFieldRenderer, NodeInfo } from "../types";

type OnIssueEventConfiguration = {
  events?: string[];
};

export const onIssueEventCustomFieldRenderer: CustomFieldRenderer = {
  render: (node: NodeInfo) => {
    const configuration = node.configuration as OnIssueEventConfiguration | undefined;

    return (
      <div className="border-t-1 border-gray-200 pt-4">
        <div className="space-y-3">
          <div>
            <span className="text-sm font-medium text-gray-700 dark:text-gray-300">Setup</span>
            <p className="text-sm text-gray-600 dark:text-gray-400 mt-1">
              Connect the Sentry integration (Install + Attach). Then trigger an issue change in Sentry (
              {configuration?.events?.length ? configuration.events.join(", ") : "created/resolved/unresolved"}).
            </p>
          </div>
        </div>
      </div>
    );
  },
};
