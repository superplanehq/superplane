import { NodeProps } from '@xyflow/react';
import CustomBarHandle from './handle';
import { EventSourceNodeType } from '@/canvas/types/flow';


export default function EventSourceNode( props : NodeProps<EventSourceNodeType>) {
  return (
    <div className={`bg-white roundedg shadow-md border ${props.selected ? 'ring-2 ring-blue-500' : 'border-gray-200'}`}>
      <div className="flex items-center p-3 bg-[#24292e] text-white rounded-tg">
        <span className="mr-2">
          <svg viewBox="0 0 16 16" width="16" height="16" fill="currentColor">
            <path fillRule="evenodd" d="M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82.64-.18 1.32-.27 2-.27.68 0 1.36.09 2 .27 1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8.013 8.013 0 0016 8c0-4.42-3.58-8-8-8z"></path>
          </svg>
        </span>
        <span className="font-semibold">{props.data.repoName}</span>
        {props.selected && <div className="absolute top-0 right-0 w-3 h-3 bg-blue-500 rounded-full m-1"></div>}
      </div>
      <div className="p-4">
        <div className="mb-3">
          <a href={props.data.repoUrl} target="_blank" rel="noopener noreferrer" className="text-blue-600 hover:text-blue-800 break-all">
            {props.data.repoUrl}
          </a>
        </div>
        <div>
          <h4 className="text-sm font-medium text-gray-700 mb-2">Last Event</h4>
          <div className="bg-gray-50 border border-gray-200 rounded p-3">
            <div className="flex justify-between mb-1">
              <span className="text-sm text-gray-600">Event Type:</span>
              <span className="text-sm font-medium">{props.data.lastEvent.type}</span>
            </div>
            <div className="flex justify-between mb-1">
              <span className="text-sm text-gray-600">Release:</span>
              <span className="text-sm font-medium">{props.data.lastEvent.release}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-sm text-gray-600">Timestamp:</span>
              <span className="text-sm font-medium">{props.data.lastEvent.timestamp}</span>
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
