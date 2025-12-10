import React from "react";
import { ApprovalItem, ApprovalItemProps } from "../approvalItem";
import { ItemGroup } from "../item";

export interface ApprovalGroupProps {
  approvals: ApprovalItemProps[];
  awaitingApproval: boolean;
}

export const ApprovalGroup: React.FC<ApprovalGroupProps> = ({ approvals, awaitingApproval }) => {
  if (!awaitingApproval) {
    return (
      <div className="flex items-center justify-center px-2 py-4 rounded-md bg-gray-50 border border-dashed border-gray-300 m-3 mt-4">
        <span className="text-sm text-gray-400">Awaiting events for approval</span>
      </div>
    );
  }

  return (
    <ItemGroup className="w-full p-3">
      {approvals.map((approval, index) => (
        <React.Fragment key={`${approval.title}-${index}`}>
          <ApprovalItem {...approval} />
        </React.Fragment>
      ))}
    </ItemGroup>
  );
};
