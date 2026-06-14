import type { ComponentBaseMapper } from "../types";
import { coolifyBaseMapper } from "./base";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  listApplications: coolifyBaseMapper,
  listServices: coolifyBaseMapper,
  controlApplication: coolifyBaseMapper,
  controlService: coolifyBaseMapper,
  deployApplication: coolifyBaseMapper,
};
