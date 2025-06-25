import { useState, ReactNode, useMemo } from 'react';
import type { NodeProps } from '@xyflow/react';
import CustomBarHandle from './handle';
import { StageNodeType } from '@/canvas/types/flow';
import { useCanvasStore } from '../../store/canvasStore';
import { SuperplaneExecution } from '@/api-client';

export default function StageNode(props: NodeProps<StageNodeType>) {
  const [showOverlay, setShowOverlay] = useState(false);
  const { selectStageId } = useCanvasStore();

  // Filter events by their state
  const pendingEvents = useMemo(() => 
    props.data.queues?.filter(event => event.state === 'STATE_PENDING') || [], 
    [props.data.queues]
  );

  const waitingEvents = useMemo(() => 
    props.data.queues?.filter(event => event.state === 'STATE_WAITING') || [], 
    [props.data.queues]
  );
  
  const processedEvents = useMemo(() => 
    props.data.queues?.filter(event => event.state === 'STATE_PROCESSED') || [], 
    [props.data.queues]
  );

  const allExecutions = useMemo(() => 
    props.data.queues?.flatMap(event => event.execution as SuperplaneExecution)
      .filter(execution => execution)
      .sort((a, b) => new Date(b?.createdAt || '').getTime() - new Date(a?.createdAt || '').getTime()) || [], 
    [props.data.queues]
  );

  const allFinishedExecutions = useMemo(() =>
    allExecutions.filter(execution => execution?.finishedAt), 
    [allExecutions]
  );

  const executionRunning = useMemo(() => 
    allExecutions.some(execution => execution.state === 'STATE_STARTED'), 
    [allExecutions]
  );

  const outputs = useMemo(() => {
    const lastFinishedExecution = allFinishedExecutions.at(0);

    return props.data.outputs.map(output => {
      const executionOutput = lastFinishedExecution?.outputs?.find(
        executionOutput => executionOutput.name === output.name
      );
      return {
        key: output.name,
        value: executionOutput?.value || '—',
        required: !!output.required
      };
    });
  }, [props.data.outputs, allFinishedExecutions]);

  // Get status icon and color
  const getStatusIcon = () => {
    switch (props.data.status?.toLowerCase()) {
      case 'passed':
      case 'success':
        return <span className="material-symbols-outlined fill text-green-600">check_circle</span>;
      case 'failed':
      case 'error':
        return <span className="material-symbols-outlined fill text-red-600">cancel</span>;
      case 'queued':
        return <span className="material-symbols-outlined fill text-orange-600">queue</span>;
      case 'running':
        return <span className="rounded-full bg-blue-500 w-[22px] h-[22px] flex items-center justify-center">
          <span className="text-white text-xs animate-pulse">●</span>
        </span>;
      default:
        return <span className="material-symbols-outlined fill text-gray-400">radio_button_unchecked</span>;
    }
  };

  // Get status background color class
  const getStatusBgClass = () => {
    switch (props.data.status?.toLowerCase()) {
      case 'passed':
      case 'success':
        return 'bg-green-50 border-green-200';
      case 'failed':
      case 'error':
        return 'bg-red-50 border-red-200';
      case 'running':
        return 'bg-blue-50 border-blue-200';
      case 'queued':
        return 'bg-yellow-50 border-yellow-200';
      default:
        return 'bg-green-50 border-green-200';
    }
  };

  return (
    <div 
      className={`bg-white rounded-lg border ${props.selected ? 'border-blue-500 ring-2 ring-blue-200' : 'border-gray-200'} relative`}
      style={{ 
        width: 320,
        boxShadow: '0 4px 12px rgba(128,128,128,0.20)' 
      }}
    >
      {/* Selected state icon overlay */}
      {props.selected && (
        <div className="absolute -top-12 left-1/2 -translate-x-1/2 flex gap-2 bg-white shadow-lg rounded-lg px-2 py-1 border z-10">
          <button 
            className="hover:bg-gray-100 text-gray-600 px-2 py-1 rounded leading-none" 
            title="Start Run"
          >
            <span className="material-symbols-outlined" style={{fontSize:20}}>play_arrow</span>
          </button>
          <button 
            className="hover:bg-gray-100 text-gray-600 px-2 py-1 rounded leading-none" 
            title="View Code"
            onClick={() => setShowOverlay(true)}
          >
            <span className="material-symbols-outlined" style={{fontSize:20}}>code</span>
          </button>
          <button 
            className="hover:bg-gray-100 text-gray-600 px-2 py-1 rounded leading-none" 
            title="Edit Triggers"
          >
            <span className="material-symbols-outlined" style={{fontSize:20}}>bolt</span>
          </button>
          <button 
            className="hover:bg-red-100 hover:text-red-600 text-gray-600 px-2 py-1 rounded leading-none" 
            title="Delete Stage"
          >
            <span className="material-symbols-outlined" style={{fontSize:20}}>delete</span>
          </button>
        </div>
      )}

      {/* Modal overlay for View Code */}
      <OverlayModal open={showOverlay} onClose={() => setShowOverlay(false)}>
        <h2 style={{ fontSize: 22, fontWeight: 700, marginBottom: 16 }}>Stage Code</h2>
        <div style={{ color: '#444', fontSize: 16, lineHeight: 1.7 }}>
          Lorem ipsum dolor sit amet, consectetur adipiscing elit. Suspendisse et urna fringilla, tincidunt nulla nec, dictum erat.
        </div>
      </OverlayModal>

      {/* Header */}
      <div className="p-3 flex justify-between items-center">
        <div className="flex items-center">
          <span className="material-symbols-outlined mr-2 text-gray-600">rocket_launch</span>
          <p className="mb-0 font-semibold text-gray-900">{props.data.label}</p>
        </div>
        <div className='flex items-center gap-2'>
          {/* Health status indicator */}
          <div className="rounded-full bg-green-500 w-3 h-3 border-2 border-green-200" title="Healthy"></div>
          
          {/* Menu button */}
          <button 
            onClick={() => selectStageId(props.id)} 
            className="p-1 rounded hover:bg-gray-100 transition" 
            title="More actions"
          >
            <span className="material-symbols-outlined text-gray-500">more_vert</span>
          </button>
        </div>
      </div>
      
      {/* Last Run Section */}
      <div className={`p-3 ${getStatusBgClass()} w-full border-t min-w-0 text-ellipsis overflow-hidden`}>
        <div className="flex items-center w-full justify-between mb-2">
          <div className="uppercase text-xs font-medium text-gray-600">Last run</div>
          <div className="text-xs text-gray-500">{props.data.timestamp}</div>
        </div>

        <div className="flex items-center mb-2">
          {getStatusIcon()}
          <img alt="Favicon" className="w-4 h-4 mx-2" src="/favicon.ico"/>
          <a href="#" className="min-w-0 font-medium text-sm flex items-center hover:underline truncate">
            {props.data.status} - Latest execution
          </a>
        </div>
        
        {/* Output badges */}
        <div className="flex flex-wrap gap-1 mt-2">
          {outputs.map((output, index) => (
            <span 
              key={index}
              className={`bg-gray-100 text-gray-700 text-xs px-2 py-1 rounded-full max-w-32 truncate ${
                output.required ? 'border border-gray-300 font-medium' : ''
              }`}
            >
              {output.key}: {output.value}
            </span>
          ))}
        </div>
      </div>

      {/* Queue Section */}
      <div className="p-3 pt-2 w-full">
        <div className="uppercase text-xs font-medium text-gray-600 mb-2">QUEUE</div>
        <div className="w-full">
          {/* Pending Events */}
          {pendingEvents.length > 0 ? (
            <div className='flex items-center w-full p-2 bg-gray-100 rounded mb-1'>
              <div className="rounded-full bg-amber-100 text-amber-600 w-6 h-6 mr-2 flex items-center justify-center">
                <span className="material-symbols-outlined text-sm">pending</span>
              </div>
              <div className="min-w-0 text-sm font-medium flex items-center truncate">
                <div className='truncate'>Pending: {new Date(pendingEvents[0].createdAt || '').toLocaleString()}</div>
              </div>
            </div>
          ) : null}

          {/* Waiting Events */}
          {waitingEvents.length > 0 ? (
            <div className='flex items-center w-full p-2 bg-blue-50 rounded mb-1'>
              <div className="rounded-full bg-blue-100 text-blue-600 w-6 h-6 mr-2 flex items-center justify-center">
                <span className="material-symbols-outlined text-sm">how_to_reg</span>
              </div>
              <div className="flex-1 min-w-0">
                <div className="text-sm font-medium truncate">
                  Waiting for approval
                </div>
                <div className="text-xs text-gray-600">
                  {new Date(waitingEvents[0].createdAt!).toLocaleString()}
                </div>
              </div>
              <button 
                onClick={() => !executionRunning && props.data.approveStageEvent(waitingEvents[0])}
                disabled={executionRunning}
                className="ml-2 px-2 py-1 text-xs bg-blue-600 text-white rounded hover:bg-blue-700 disabled:bg-gray-400 disabled:cursor-not-allowed"
              >
                Approve
              </button>
            </div>
          ) : null}

          {/* Show empty state */}
          {pendingEvents.length === 0 && waitingEvents.length === 0 && processedEvents.length === 0 && (
            <div className="text-sm text-gray-500 italic">No items in queue</div>
          )}
        </div>
      </div>

      <CustomBarHandle type="target" connections={props.data.connections} conditions={props.data.conditions}/>
      <CustomBarHandle type="source"/>
    </div>
  );
}

interface OverlayModalProps {
  open: boolean;
  onClose: () => void;
  children: ReactNode;
}

function OverlayModal({ open, onClose, children }: OverlayModalProps) {
  if (!open) return null;
  return (
    <div className="modal is-open" aria-hidden={!open} style={{position:'fixed',top:0,left:0,right:0,bottom:0,zIndex:999999}}>
      <div className="modal-overlay" style={{position:'fixed',top:0,left:0,right:0,bottom:0,background:'rgba(40,50,50,0.6)',zIndex:999999}} onClick={onClose} />
      <div className="modal-content" style={{position:'fixed',top:'50%',left:'50%',transform:'translate(-50%, -50%)',zIndex:1000000,background:'#fff',borderRadius:8,boxShadow:'0 6px 40px rgba(0,0,0,0.18)',maxWidth:600,width:'90vw',padding:32}}>
        <button onClick={onClose} style={{position:'absolute',top:8,right:12,background:'none',border:'none',fontSize:26,color:'#888',cursor:'pointer'}} aria-label="Close">×</button>
        {children}
      </div>
    </div>
  );
}