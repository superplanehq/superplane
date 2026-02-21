import { ComponentBaseMapper } from "../types";
import { hetznerBaseMapper } from "./base";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  createServer: hetznerBaseMapper,
  deleteServer: hetznerBaseMapper,
  createLoadBalancer: hetznerBaseMapper,
  deleteLoadBalancer: hetznerBaseMapper,
};
