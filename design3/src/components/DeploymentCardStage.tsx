import React, { useState, useCallback, useMemo } from 'react';
import { Handle, Position, NodeProps } from '@xyflow/react';
import Tippy from '@tippyjs/react';
import { DeploymentCardStageProps } from '../types';

/**
 * OverlayModal component for displaying code or other content
 */
const OverlayModal: React.FC<{
  open: boolean;
  onClose: () => void;
  children: React.ReactNode;
}> = ({ open, onClose, children }) => {
  if (!open) return null;

  return (
    <div 
      className="fixed inset-0 z-50 flex items-center justify-center bg-black bg-opacity-50"
      onClick={onClose}
      role="dialog"
      aria-modal="true"
      aria-labelledby="modal-title"
    >
      <div 
        className="bg-white rounded-lg p-6 max-w-2xl w-full mx-4 max-h-[80vh] overflow-auto"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex justify-between items-start mb-4">
          <h2 id="modal-title" className="text-xl font-semibold text-gray-900">
            Stage Code
          </h2>
          <button
            type="button"
            onClick={onClose}
            className="text-gray-400 hover:text-gray-600 transition-colors"
            aria-label="Close modal"
          >
            <span className="material-symbols-outlined">close</span>
          </button>
        </div>
        {children}
      </div>
    </div>
  );
};

/**
 * DeploymentCardStage component following SaaS guidelines
 * - Uses TypeScript with proper interfaces
 * - Implements accessibility features
 * - Follows responsive design principles
 * - Uses Tailwind utility classes
 * - Includes proper error handling
 */
export const DeploymentCardStage: React.FC<NodeProps<DeploymentCardStageProps['data']>> = ({
  data,
  selected,
  id,
}) => {
  const [showOverlay, setShowOverlay] = useState(false);

  const handleAction = useCallback(
    (action: string) => {
      if (action === 'code') {
        setShowOverlay(true);
      }
      // Additional actions can be handled here
      console.log(`Action: ${action} on node: ${id}`);
    },
    [id]
  );

  const handleDelete = useCallback(() => {
    console.log(`Delete node: ${id}`);
  }, [id]);

  // Memoized node style to prevent unnecessary re-renders
  const nodeStyle = useMemo(
    () => ({
      width: data.style?.width || 320,
      boxShadow: '0 4px 12px rgba(0, 0, 0, 0.1)',
    }),
    [data.style?.width]
  );

  // Get status colors and icons
  const getStatusConfig = useCallback((status: string) => {
    switch (status?.toLowerCase()) {
      case 'success':
      case 'passed':
        return {
          bgColor: 'bg-success-50',
          borderColor: 'border-success-200',
          textColor: 'text-success-700',
          icon: 'check_circle',
          iconColor: 'text-success-500',
        };
      case 'failed':
        return {
          bgColor: 'bg-error-50',
          borderColor: 'border-error-200',
          textColor: 'text-error-700',
          icon: 'cancel',
          iconColor: 'text-error-500',
        };
      case 'running':
        return {
          bgColor: 'bg-primary-50',
          borderColor: 'border-primary-200',
          textColor: 'text-primary-700',
          icon: 'sync',
          iconColor: 'text-primary-500 animate-spin',
        };
      case 'queued':
        return {
          bgColor: 'bg-warning-50',
          borderColor: 'border-warning-200',
          textColor: 'text-warning-700',
          icon: 'queue',
          iconColor: 'text-warning-500',
        };
      default:
        return {
          bgColor: 'bg-gray-50',
          borderColor: 'border-gray-200',
          textColor: 'text-gray-700',
          icon: 'help',
          iconColor: 'text-gray-500',
        };
    }
  }, []);

  const statusConfig = getStatusConfig(data.status || 'pending');

  return (
    <div 
      className={`bg-white rounded-lg border-2 ${
        selected ? 'border-primary-500 ring-2 ring-primary-200' : 'border-gray-200'
      } relative transition-all duration-200 hover:shadow-lg`}
      style={nodeStyle}
      role="article"
      aria-label={`Deployment stage: ${data.label}`}
    >
      {/* Action buttons when selected */}
      {selected && (
        <div className="absolute -top-12 left-1/2 transform -translate-x-1/2 flex gap-1 bg-white shadow-lg rounded-lg px-2 py-1 border border-gray-200 z-10">
          <Tippy content="Start a run for this stage" placement="top">
            <button
              type="button"
              className="hover:bg-gray-100 text-gray-600 px-2 py-1 rounded-md transition-colors focus:outline-none focus:ring-2 focus:ring-primary-500"
              onClick={() => handleAction('run')}
              aria-label="Start run"
            >
              <span className="material-symbols-outlined text-sm">play_arrow</span>
            </button>
          </Tippy>

          <Tippy content="View code for this stage" placement="top">
            <button
              type="button"
              className="hover:bg-gray-100 text-gray-600 px-2 py-1 rounded-md transition-colors focus:outline-none focus:ring-2 focus:ring-primary-500"
              onClick={() => handleAction('code')}
              aria-label="View code"
            >
              <span className="material-symbols-outlined text-sm">code</span>
            </button>
          </Tippy>

          <Tippy content="Edit triggers for this stage" placement="top">
            <button
              type="button"
              className="hover:bg-gray-100 text-gray-600 px-2 py-1 rounded-md transition-colors focus:outline-none focus:ring-2 focus:ring-primary-500"
              onClick={() => handleAction('edit')}
              aria-label="Edit triggers"
            >
              <span className="material-symbols-outlined text-sm">bolt</span>
            </button>
          </Tippy>

          <Tippy content="Delete this stage" placement="top">
            <button
              type="button"
              className="hover:bg-error-100 hover:text-error-600 text-gray-600 px-2 py-1 rounded-md transition-colors focus:outline-none focus:ring-2 focus:ring-error-500"
              onClick={handleDelete}
              aria-label="Delete stage"
            >
              <span className="material-symbols-outlined text-sm">delete</span>
            </button>
          </Tippy>
        </div>
      )}

      {/* Modal overlay for viewing code */}
      <OverlayModal open={showOverlay} onClose={() => setShowOverlay(false)}>
        <div className="text-gray-700 leading-relaxed">
          <p className="mb-4">
            This would show the actual code configuration for this deployment stage.
          </p>
          <pre className="bg-gray-100 p-4 rounded-md text-sm overflow-auto">
            {`# Deployment Configuration
version: v1.0
agent:
  machine:
    type: e1-standard-2
    os_image: ubuntu2004
blocks:
  - name: "${data.label}"
    task:
      jobs:
        - name: deploy
          commands:
            - checkout
            - cache restore
            - npm install
            - npm run build
            - npm run deploy`}
          </pre>
        </div>
      </OverlayModal>

      {/* Header section */}
      <div className="p-4 flex justify-between items-center border-b border-gray-200">
        <div className="flex items-center">
          <span className="material-symbols-outlined mr-2 text-gray-600">
            rocket_launch
          </span>
          <h3 className="font-semibold text-gray-900">{data.label}</h3>
        </div>

        {/* Health check indicator */}
        {data.hasHealthCheck && (
          <div className="flex items-center">
            <Tippy 
              content={`Health check: ${data.healthCheckStatus}. Last check run 2 hours ago`}
              placement="top"
            >
              <div
                className={`w-3 h-3 rounded-full border-2 ${
                  data.healthCheckStatus === 'healthy'
                    ? 'bg-success-500 border-success-200'
                    : data.healthCheckStatus === 'unhealthy'
                    ? 'bg-error-500 border-error-200'
                    : 'bg-warning-500 border-warning-200'
                }`}
                role="status"
                aria-label={`Health status: ${data.healthCheckStatus}`}
              />
            </Tippy>
          </div>
        )}
      </div>

      {/* Status section */}
      <div 
        className={`p-4 ${statusConfig.bgColor} border-b border-gray-200`}
      >
        <div className="flex items-center justify-between mb-2">
          <span className="text-xs font-medium text-gray-500 uppercase tracking-wide">
            Last Run
          </span>
          <span className="text-xs text-gray-500">2 hours ago</span>
        </div>

        <div className="flex items-center mb-3">
          <span className={`material-symbols-outlined mr-2 ${statusConfig.iconColor}`}>
            {statusConfig.icon}
          </span>
          <span className="font-medium text-sm text-gray-900">
            BUG-213 When clicking on the...
          </span>
        </div>

        {/* Labels/tags */}
        <div className="flex flex-wrap gap-2">
          <span className="bg-gray-100 text-gray-700 text-xs px-2 py-1 rounded-full">
            code: 1042a82
          </span>
          <span className="bg-gray-100 text-gray-700 text-xs px-2 py-1 rounded-full">
            image: v.4.1.3
          </span>
          <span className="bg-gray-100 text-gray-700 text-xs px-2 py-1 rounded-full">
            terraform: v.2.9.2
          </span>
        </div>
      </div>

      {/* Queue section */}
      <div className="p-4">
        <h4 className="text-xs font-medium text-gray-500 uppercase tracking-wide mb-2">
          Queue
        </h4>
        <div className="space-y-2">
          <div className="flex items-center p-2 bg-gray-50 rounded-md">
            <Tippy content="Need manual approval" placement="top">
              <div className="w-6 h-6 bg-warning-100 text-warning-600 rounded-full flex items-center justify-center mr-2">
                <span className="material-symbols-outlined text-sm">how_to_reg</span>
              </div>
            </Tippy>
            <span className="text-sm text-gray-700 font-medium truncate">
              Waiting for approval
            </span>
          </div>
        </div>
      </div>

      {/* React Flow handles */}
      <Handle
        type="target"
        position={Position.Left}
        className="w-3 h-3 !bg-primary-500 !border-2 !border-white"
        aria-label="Input connection point"
      />
      <Handle
        type="source"
        position={Position.Right}
        className="w-3 h-3 !bg-primary-500 !border-2 !border-white"
        aria-label="Output connection point"
      />
    </div>
  );
};