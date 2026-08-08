import type { ComponentBaseMapper } from "../types";
import { dokployBaseMapper } from "./base";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  listApplications: dokployBaseMapper,
  deployApplication: dokployBaseMapper,
};
