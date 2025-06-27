import { useState, ReactNode, useMemo } from 'react';
import type { NodeProps } from '@xyflow/react';
import CustomBarHandle from './handle';
import { StageNodeType } from '@/canvas/types/flow';
import { useCanvasStore } from '../../store/canvasStore';
import { SuperplaneExecution, SuperplaneExecutionState } from '@/api-client';

// Define the data type for the deployment card
// Using Record<string, unknown> to satisfy ReactFlow's Node constraint
export default function StageNode(props: NodeProps<StageNodeType>) {
  const [showOverlay, setShowOverlay] = useState(false);
  const { selectStageId } = useCanvasStore()

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
    allExecutions
        .filter(execution => execution?.finishedAt)
    , [allExecutions]
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
      )
      return {
        key: output.name,
        value: executionOutput?.value || '—',
        required: !!output.required
      }
    })
  }, [props.data.outputs, allFinishedExecutions])

  const getStatusColor = (status: SuperplaneExecutionState) => {
    switch (status) {
      case 'STATE_STARTED':
      case 'STATE_FINISHED':
        return 'bg-green-100 text-green-800';
      case 'STATE_UNKNOWN':
        return 'bg-red-100 text-red-800';
      case 'STATE_PENDING':
        return 'bg-yellow-100 text-yellow-800';
      default:
        return 'bg-gray-100 text-gray-800';
    }
  };

  const isRunning = executionRunning || props.data.status?.toLowerCase() === 'running';
  
  return (
    <div
      onClick={() => selectStageId(props.id)}
      className={`bg-white rounded-lg shadow-lg border-2 ${props.selected ? 'border-blue-400' : 'border-gray-200'} relative`}
      style={{ width: '320px', boxShadow: 'rgba(128, 128, 128, 0.2) 0px 4px 12px' }}
    >
      {/* Modal overlay for View Code */}
      <OverlayModal open={showOverlay} onClose={() => setShowOverlay(false)}>
        <h2 style={{ fontSize: 22, fontWeight: 700, marginBottom: 16 }}>Stage Code</h2>
        <div style={{ color: '#444', fontSize: 16, lineHeight: 1.7 }}>
          Lorem ipsum dolor sit amet, consectetur adipiscing elit. Suspendisse et urna fringilla, tincidunt nulla nec, dictum erat. Etiam euismod, justo id facilisis dictum, urna massa dictum erat, eget dictum urna massa id justo. Praesent nec facilisis urna. Pellentesque habitant morbi tristique senectus et netus et malesuada fames ac turpis egestas.
        </div>
      </OverlayModal>

      {/* Header Section */}
      <div className="px-4 py-4 flex justify-between items-center">
        <div className="flex items-center">
          <span className="material-symbols-outlined mr-2 text-gray-700">rocket_launch</span>
          <p className="mb-0 font-bold ml-1 text-gray-900">{props.data.label}</p>
        </div>
        <div className="flex items-center">
          <span className="rounded-full bg-green-500 w-3 h-3 border border-green-300"></span>
        </div>
      </div>

      {/* Last Run Section */}
      <div className="px-3 py-3 bg-blue-50 border-t border-blue-200 w-full">
        <div className="flex items-center w-full justify-between mb-2">
          <div className="text-xs font-medium text-gray-700 uppercase tracking-wide">Last run</div>
          <div className="text-xs text-gray-600">
            {isRunning ? 'Deploying now' : props.data.timestamp || 'No recent runs'}
          </div>
        </div>
        
        {/* Current Execution Display */}
        <div>
          <div className="flex items-center mb-1">
            <span className={`rounded-full w-6 h-6 border border-blue-200 text-center mr-2 flex items-center justify-center ${isRunning ? 'bg-blue-500' : 'bg-gray-400'}`}>
              {isRunning && (
                <span className="text-white text-sm animate-spin">⟳</span>
              )}
            </span>
            <a 
              href="#" 
              className="min-w-0 font-semibold text-sm flex items-center hover:underline truncate text-gray-900"
              onClick={() => selectStageId(props.id)}
            >
              {props.data.label || 'Stage execution'}
            </a>
          </div>
          
          {/* Output Tags */}
          <div className="flex flex-wrap gap-1 mt-2">
            {outputs.slice(0, 4).map((output, index) => (
              <span 
                key={index}
                className={`text-xs px-2 py-1 rounded-full ${
                  output.required 
                    ? 'bg-gray-200 text-gray-800 border border-gray-300 font-medium' 
                    : 'bg-gray-100 text-gray-700'
                }`}
              >
                {output.key}: {output.value}
              </span>
            ))}
          </div>
        </div>
      </div>

      {/* Queue Section */}
      <div className="px-3 pt-2 pb-0 w-full">
        <div className="w-full text-left text-xs font-medium text-gray-700 uppercase tracking-wide mb-1">Queue</div>
        
        <div className="w-full pt-1 pb-6">
          {/* Pending Events */}
          {pendingEvents.length > 0 && (
            <div className="flex items-center w-full p-2 bg-gray-100 rounded-lg mt-1">
              <div className="rounded-full bg-[var(--lightest-orange)] text-[var(--dark-orange)] w-6 h-6 mr-2 flex items-center justify-center">
                <span className="material-symbols-outlined" style={{ fontSize: '19px' }}>how_to_reg</span>
              </div>
              <a 
                href="#" 
                className="min-w-0 font-semibold text-sm flex items-center hover:underline"
              >
                <div className="truncate">Deploy: Pending ({pendingEvents.length})</div>
              </a>
            </div>
          )}

          {/* Waiting Events */}
          {waitingEvents.length > 0 && (
            <div className="flex items-center w-full p-2 bg-gray-100 rounded-lg mt-1">
              <div className="rounded-full bg-[var(--lightest-orange)] text-[var(--dark-orange)] w-6 h-6 mr-2 flex items-center justify-center">
                <span className="material-symbols-outlined" style={{ fontSize: '19px' }}>how_to_reg</span>
              </div>
              <a 
                href="#" 
                className="min-w-0 font-semibold text-sm flex items-center hover:underline"
              >
                <div className="truncate">Scale: Waiting Approval ({waitingEvents.length})</div>
              </a>
            </div>
          )}

          {/* Show empty state when no queue items */}
          {!pendingEvents.length && !waitingEvents.length && (
            <div className="text-sm text-gray-500 italic py-2">No queue activity</div>
          )}
        </div>
      </div>

      {/* Custom Handles */}
      <CustomBarHandle type="target" connections={props.data.connections} conditions={props.data.conditions}/>
      <CustomBarHandle type="source"/>
    </div>
  );
};

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