import { NodeProps } from '@xyflow/react';
import CustomBarHandle from './handle';
import { EventSourceNodeType } from '@/canvas/types/flow';


export default function EventSourceNode( props : NodeProps<EventSourceNodeType>) {
  return (
    <div className={`bg-white min-w-70 roundedg shadow-md border ${props.selected ? 'ring-2 ring-blue-500' : 'border-gray-200'}`}>
      <div className="flex items-center p-3 bg-[#24292e] text-white rounded-tg">
        <span className="font-semibold">Event Source</span>
        {props.selected && <div className="absolute top-0 right-0 w-3 h-3 bg-blue-500 rounded-full m-1"></div>}
      </div>
      <div className="p-4">
        <div className="mb-3">
          <a href={props.data.name} target="_blank" rel="noopener noreferrer" className="text-blue-600 hover:text-blue-800 break-all">
            {props.data.name}
          </a>
        </div>
        <div>
          <h4 className="text-sm font-medium text-gray-700 mb-2">Last Event</h4>
          <div className="bg-gray-50 border border-gray-200 rounded p-3">
            <div className="flex justify-between">
              <span className="text-sm text-gray-600">Timestamp:</span>
              <span className="text-sm font-medium">{props.data.timestamp}</span>
            </div>
          </div>
        </div>
      </div>
      <CustomBarHandle 
        type="source" 
      />
    </div>
  );
}
