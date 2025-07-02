import { NodeProps } from '@xyflow/react';
import CustomBarHandle from './handle';
import { EventSourceNodeType } from '@/canvas/types/flow';


export default function EventSourceNode( props : NodeProps<EventSourceNodeType>) {
  
  return (
    <div className={`w-100 bg-white min-w-70 roundedg shadow-md border ${props.selected ? 'ring-2 ring-blue-500' : 'border-gray-200'}`}>
      <div className="flex items-center p-5 rounded-tg border-b border-gray-200 text-lg">
        <span className="font-semibold">Event Source</span>
        {props.selected && <div className="absolute top-0 right-0 w-3 h-3 bg-blue-500 rounded-full m-1"></div>}
      </div>
      <div className="p-4">
        <div className="mb-3 w-full text-left text-lg">
          <a href={props.data.name} target="_blank" rel="noopener noreferrer" className="link text-[var(--dark-indigo)]  break-all">
            {props.data.name}
          </a>
        </div>
        <div>
          <h4 className="text-left text-normal font-medium text-gray-700 mb-5">EVENTS</h4>
          <div className="space-y-1">
          {
            props.data.events?.map((event) => (
              <div key={event.id} className="bg-gray-100 rounded p-2">
                <div className="flex justify-start items-center gap-3 overflow-hidden">
                  <span className="text-sm text-gray-600">
                    <i className="material-icons f3 fill-black rounded-full bg-[var(--washed-green)] black-60 p-1">bolt</i>
                  </span>
                  <span className="truncate">{event.id!}</span>
                </div>
              </div>
            ))
          }
          </div>
        </div>
      </div>
      <CustomBarHandle 
        type="source" 
      />
    </div>
  );
}
