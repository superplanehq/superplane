import { SuperplaneExecution, SuperplaneInputValue, SuperplaneOutputValue } from "@/api-client";
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

  const generateKeyValueMap = (keyValues: SuperplaneOutputValue[] | SuperplaneInputValue[] | undefined) => {
    if (!keyValues) {
      return {};
    }

    const map: Record<string, string> = {};
    keyValues.forEach((keyValue) => {
      if (!keyValue.value) {
        return;
      }

      map[keyValue.name!] = keyValue.value;
    });
    return map;
  };

  return (
    <>
      {
        executions.map((execution) => (
          <RunItem key={execution.id!} title={'Execution'} inputs={{}} outputs={generateKeyValueMap(execution.outputs)} status={execution.state || 'Unknown'} timestamp={execution.createdAt || 'Unknown'} />
        ))
      }
    </>
  );
};