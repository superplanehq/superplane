import { SuperplaneExecution } from "@/api-client";
import { RunItem } from "./tabs/RunItem";

interface ExecutionTimelineProps {
  executions: SuperplaneExecution[];
}

export const ExecutionTimeline = ({ 
  executions, 
}: ExecutionTimelineProps) => {
  if (executions.length === 0) {
    return (
      <div className="bg-white rounded-lg border border-gray-200">
        <div className="p-4">
          <div className="text-center py-6 text-gray-500">
            <div className="text-4xl mb-2">📊</div>
            <div className="text-sm">No recent activity</div>
          </div>
        </div>
      </div>
    );
  }

  return (
    <>
      <RunItem title="123fdsdsf" imageVersion="123" status="123" timestamp="123" extraTags="123" isHightlighted />
    </>
  );
};