import type { ComponentBaseContext, ComponentBaseMapper, ExecutionDetailsContext, SubtitleContext } from "../types";
import { baseMapper } from "./base";
import { addExecutedAt, getOutputData, imageMetadataList } from "./image_common";

export const deleteImageMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext) {
    const props = baseMapper.props(context);
    return {
      ...props,
      metadata: imageMetadataList(context.node),
    };
  },

  subtitle(context: SubtitleContext) {
    return baseMapper.subtitle(context);
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const details: Record<string, string> = {};
    addExecutedAt(details, context);

    const data = getOutputData(context);
    if (!data) return details;

    if (data.imageId) details["Image ID"] = data.imageId;
    if (data.state) details["State"] = data.state;
    if (data.deletedAt) details["Deleted At"] = new Date(data.deletedAt).toLocaleString();

    return details;
  },
};
