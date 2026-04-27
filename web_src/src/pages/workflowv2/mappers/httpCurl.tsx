import type { CustomFieldRenderer, CustomFieldRendererContext, NodeInfo } from "@/pages/workflowv2/mappers/types";
import { HttpCurlCustomField } from "./httpCurlCustomField";

export const httpCurlCustomFieldRenderer: CustomFieldRenderer = {
  render: (node: NodeInfo, context?: CustomFieldRendererContext) => {
    return <HttpCurlCustomField node={node} context={context} />;
  },
};
