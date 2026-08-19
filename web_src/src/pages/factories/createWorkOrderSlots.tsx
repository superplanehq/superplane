import { createContext, useContext } from "react";

/**
 * Storybook can seed the id of the "current user" so the create-work-order
 * composer defaults Owner to them. The live app leaves this empty, so Owner
 * still starts blank until account context wires a real current-user id.
 */
export const CreateWorkOrderDefaultOwnerSlotContext = createContext<string | null>(null);

export function useCreateWorkOrderDefaultOwnerId() {
  return useContext(CreateWorkOrderDefaultOwnerSlotContext);
}
