import { NodeProps } from '@xyflow/react';
import CustomBarHandle from './handle';
import { EventSourceNodeType } from '@/canvas/types/flow';

export default function EventSourceNode(props: NodeProps<EventSourceNodeType>) {
  // Determine the integration type based on the name/URL
  const isKubernetes = props.data.name.includes('cluster') || props.data.name.includes('kubernetes');
  const isS3 = props.data.name.includes('buckets/') || props.data.name.includes('s3');
  const isGitHub = props.data.name.includes('github') || props.data.name.includes('git');

  // Get appropriate icon based on type
  const getIcon = () => {
    if (isKubernetes) return '⚙️'; // or use kubernetes icon
    if (isS3) return '🪣'; // or use S3 icon  
    return '📁'; // default GitHub-like icon
  };

  // Get header styling based on type
  const getHeaderClass = () => {
    if (isKubernetes) return 'bg-blue-600';
    if (isS3) return 'bg-orange-500';
    return 'bg-[#24292e]'; // GitHub dark
  };

  return (
    <div 
      className={`bg-white min-w-80 rounded-lg shadow-md border ${props.selected ? 'ring-2 ring-blue-500' : 'border-gray-200'} relative`}
      style={{ 
        width: 320,
        boxShadow: '0 4px 12px rgba(128,128,128,0.20)' 
      }}
    >
      {/* Header Section */}
      <div className={`pa3 flex justify-between bb b--lightest-gray ${getHeaderClass()} text-white rounded-t-lg`}>
        <div className="flex items-center">
          <div className="d-inline-block mr-2 w-[24px] text-lg">
            {getIcon()}
          </div>
          <p className="mb0 font-semibold text-white">
            {isKubernetes ? 'Kubernetes Cluster' : isS3 ? 'S3 Bucket' : 'Repository'}
          </p>
        </div>
      </div>

      {/* Repository Info Section */}
      <div className="p-3">
        <div className="mb-2">
          <a 
            href={props.data.name} 
            target="_blank" 
            rel="noopener noreferrer" 
            className="link dark-indigo underline-hover flex items-center text-blue-600 hover:text-blue-800 break-all"
          >
            {props.data.name}
          </a>
        </div>
        
        <div className="flex items-center w-full justify-between mb-2">
          <div className="ttu f7 text-xs font-medium text-gray-500">EVENTS</div>
        </div>
      </div>

      {/* Events Section */}
      <div className="w-full p-3 pt-0">
        <div className='flex items-center w-full p-2 bg-gray-100 rounded mb-1'>
          <div className="material-symbols-outlined text-green-600 mr-2 bg-green-50 rounded-full p-1">
            bolt
          </div>
          <div className="flex-1 min-w-0">
            <div className='text-sm font-medium truncate'>Latest Event</div>
            <div className="text-xs text-gray-600 truncate">
              {props.data.timestamp}
            </div>
          </div>
        </div>

        {/* Additional event items if needed */}
        <div className='flex items-center w-full p-2 bg-gray-100 rounded mb-1'>
          <div className="material-symbols-outlined text-blue-600 mr-2 bg-blue-50 rounded-full p-1">
            sync
          </div>
          <div className="flex-1 min-w-0">
            <div className='text-sm font-medium truncate'>Configuration Sync</div>
            <div className="text-xs text-gray-600 truncate">
              2 hours ago
            </div>
          </div>
        </div>

        <div className='flex items-center w-full p-2 bg-gray-100 rounded mb-1'>
          <div className="material-symbols-outlined text-purple-600 mr-2 bg-purple-50 rounded-full p-1">
            webhook
          </div>
          <div className="flex-1 min-w-0">
            <div className='text-sm font-medium truncate'>Webhook Trigger</div>
            <div className="text-xs text-gray-600 truncate">
              5 hours ago
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