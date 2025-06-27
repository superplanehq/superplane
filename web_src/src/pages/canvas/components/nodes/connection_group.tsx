import { NodeProps } from '@xyflow/react';
import CustomBarHandle from './handle';
import { ConnectionGroupNodeType } from '@/canvas/types/flow';

export default function ConnectionGroupNode(props: NodeProps<ConnectionGroupNodeType>) {
  // Extract group by fields for display
  const groupByFields = props.data.groupBy?.fields || [];

  return (
    <div className={`bg-white min-w-70 rounded-md shadow-md border ${props.selected ? 'ring-2 ring-blue-500' : 'border-gray-200'} relative`}>
      {/* Node Header */}
      <div className="flex items-center px-3 py-2 border-b bg-gray-50 rounded-t-md">
        <span className="flex items-center justify-center w-8 h-8 bg-gray-100 rounded-full mr-2">
          <span className="material-symbols-outlined text-lg">account_tree</span>
        </span>
        <span className="font-bold text-gray-900 flex-1 text-left">Connection Group</span>
      </div>

      {props.data.name && (
        <div className="mt-2">
          <h4 className="text-sm font-medium text-gray-700 mb-1">Name</h4>
          <p className="text-sm text-gray-900 break-all">{props.data.name}</p>
        </div>
      )}

      {/* Node Content */}
      <div className="p-4">
        <div className="mb-3">
          <h4 className="text-sm font-medium text-gray-700 mb-2">Group By</h4>
          <div className="flex flex-wrap gap-2">
            {groupByFields.length > 0 ? (
              groupByFields.map((field, index) => (
                <span 
                  key={index} 
                  className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-blue-100 text-blue-800"
                >
                  {field.name}: {field.expression}
                </span>
              ))
            ) : (
              <span className="text-sm text-gray-500">No group by fields</span>
            )}
          </div>
        </div>
      </div>

      <CustomBarHandle type="target" connections={props.data.connections} />
      <CustomBarHandle type="source" />

      {props.selected && (
        <div className="absolute -top-1 -right-1 w-3 h-3 bg-blue-500 rounded-full"></div>
      )}
    </div>
  );
}
