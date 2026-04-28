import type { ComponentBaseContext, ComponentBaseMapper, ExecutionDetailsContext, SubtitleContext } from "../types";
import { baseMapper } from "./base";

export const deleteFunctionMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext) {
    return baseMapper.props(context);
  },

  subtitle(context: SubtitleContext) {
    return baseMapper.subtitle(context);
  },

  getExecutionDetails(_context: ExecutionDetailsContext): Record<string, string> {
    return {};
  },
};
