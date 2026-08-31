import { useUpdateWorkOrder, useUpdateWorkOrderAssignees } from "@/hooks/useFactoryData";
import { getApiErrorMessage } from "@/lib/errors";
import type { OrgUserDisplay } from "@/lib/orgUserDisplay";
import { getUserInitials } from "@/lib/orgUserDisplay";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { useCallback, useEffect, useRef, useState } from "react";

import type { SplitRunFooterKind } from "./splitRunFooter";

export function canEditSplitRunContent(kind: SplitRunFooterKind, canUpdate = true): boolean {
  return canUpdate && kind !== "done";
}

export function canEditSplitRunDescription(kind: SplitRunFooterKind, canUpdate = true): boolean {
  return canUpdate && kind === "draft";
}

export function useSplitRunWorkOrderEdits(args: {
  organizationId?: string;
  factoryId?: string;
  orderId?: string;
  canUpdate?: boolean;
  title: string;
  description: string;
  owner: OrgUserDisplay;
  assigneeIds: string[];
  footerKind: SplitRunFooterKind;
}) {
  const live = Boolean(args.organizationId && args.factoryId && args.orderId);
  const canEdit = canEditSplitRunContent(args.footerKind, args.canUpdate);
  const canEditDescription = canEditSplitRunDescription(args.footerKind, args.canUpdate);
  const updateWorkOrder = useUpdateWorkOrder(args.organizationId ?? "", args.factoryId ?? "");
  const updateAssignees = useUpdateWorkOrderAssignees(args.organizationId ?? "", args.factoryId ?? "");

  const [title, setTitle] = useState(args.title);
  const [description, setDescription] = useState(args.description);
  const [owner, setOwner] = useState(args.owner);
  const [assigneeIds, setAssigneeIds] = useState(args.assigneeIds);
  const descriptionSaved = useRef(false);
  const latestDescription = useRef(args.description);
  latestDescription.current = args.description;

  useEffect(() => {
    setTitle(args.title);
  }, [args.title]);

  useEffect(() => {
    descriptionSaved.current = false;
    setDescription(latestDescription.current);
  }, [args.orderId]);

  useEffect(() => {
    if (descriptionSaved.current) {
      return;
    }
    setDescription(args.description);
  }, [args.description]);

  const ownerId = args.owner.id;
  const ownerName = args.owner.name;
  const ownerInitials = args.owner.initials;
  const ownerAvatarUrl = args.owner.avatarUrl;
  const assigneeKey = args.assigneeIds.join("\0");

  useEffect(() => {
    setOwner({
      id: ownerId,
      name: ownerName,
      initials: ownerInitials,
      avatarUrl: ownerAvatarUrl,
    });
  }, [ownerAvatarUrl, ownerId, ownerInitials, ownerName]);

  useEffect(() => {
    setAssigneeIds(assigneeKey === "" ? [] : assigneeKey.split("\0"));
  }, [assigneeKey]);

  const saveTitle = useCallback(
    async (next: string) => {
      const previous = title;
      setTitle(next);
      if (!live || !args.orderId) {
        return;
      }
      try {
        await updateWorkOrder.mutateAsync({ orderId: args.orderId, title: next });
        showSuccessToast("Task title updated.");
      } catch (error) {
        setTitle(previous);
        showErrorToast(getApiErrorMessage(error, "Failed to update the title"));
      }
    },
    [args.orderId, live, title, updateWorkOrder],
  );

  const saveDescription = useCallback(
    async (next: string) => {
      const previous = description;
      descriptionSaved.current = true;
      setDescription(next);
      if (!live || !args.orderId) {
        return;
      }
      try {
        await updateWorkOrder.mutateAsync({ orderId: args.orderId, description: next });
        showSuccessToast("Task description updated.");
      } catch (error) {
        descriptionSaved.current = false;
        setDescription(previous);
        showErrorToast(getApiErrorMessage(error, "Failed to update the description"));
        throw error;
      }
    },
    [args.orderId, description, live, updateWorkOrder],
  );

  const saveOwner = useCallback(
    async (nextIds: string[]) => {
      const previousIds = assigneeIds;
      const previousOwner = owner;
      setAssigneeIds(nextIds);
      if (!live || !args.orderId) {
        return;
      }
      try {
        const order = await updateAssignees.mutateAsync({ orderId: args.orderId, assigneeIds: nextIds });
        const nextOwner = order.assignees?.[0];
        if (nextOwner?.id) {
          const name = nextOwner.name?.trim() || previousOwner.name;
          setOwner({
            id: nextOwner.id,
            name,
            initials: getUserInitials(name) || previousOwner.initials,
          });
        } else {
          setOwner(previousOwner);
        }
        showSuccessToast("Owner updated.");
      } catch (error) {
        setAssigneeIds(previousIds);
        setOwner(previousOwner);
        showErrorToast(getApiErrorMessage(error, "Failed to update the owner"));
        throw error;
      }
    },
    [args.orderId, assigneeIds, live, owner, updateAssignees],
  );

  return {
    title,
    description,
    owner,
    assigneeIds,
    canEdit,
    canEditDescription,
    titleBusy: updateWorkOrder.isPending,
    descriptionBusy: updateWorkOrder.isPending,
    ownerBusy: updateAssignees.isPending,
    saveTitle,
    saveDescription,
    saveOwner,
  };
}
