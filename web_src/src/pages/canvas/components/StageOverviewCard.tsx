interface StageOverviewCardProps {
  totalEvents: number;
  pendingCount: number;
  waitingCount: number;
  processedCount: number;
}

export const StageOverviewCard = ({ 
  totalEvents, 
  pendingCount, 
  waitingCount, 
  processedCount 
}: StageOverviewCardProps) => {
  return (
    <div className="bg-white rounded-lg border border-gray-200 p-4">
      <h3 className="text-lg font-semibold text-gray-900 mb-3">Stage Overview</h3>
      <div className="grid grid-cols-2 gap-4 text-sm">
        <div>
          <span className="text-gray-500">Total Events</span>
          <div className="font-medium text-gray-900">{totalEvents}</div>
        </div>
        <div>
          <span className="text-gray-500">Pending</span>
          <div className="font-medium text-amber-600">{pendingCount}</div>
        </div>
        <div>
          <span className="text-gray-500">Waiting</span>
          <div className="font-medium text-blue-600">{waitingCount}</div>
        </div>
        <div>
          <span className="text-gray-500">Processed</span>
          <div className="font-medium text-green-600">{processedCount}</div>
        </div>
      </div>
    </div>
  );
};