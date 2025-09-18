import React from 'react';
import { twMerge } from 'tailwind-merge';
import { useCanvasStore } from '../../store/canvasStore';

interface NodeContentWrapperProps {
  className?: string;
  children: React.ReactNode;
  nodeId?: string;
}

/**
 * Wrapper component that handles click event propagation for React Flow node content.
 * Automatically prevents propagation for interactive elements while allowing it for the container.
 * When interactive elements are clicked, it also sets the focused node after a brief delay.
 *
 * Interactive elements that will stop propagation:
 * - button, input, span, textarea, select
 * - Elements with onclick handlers
 *
 * @param nodeId - The node ID to set as focused when interactive elements are clicked
 */
export function NodeContentWrapper({ className, children, nodeId }: NodeContentWrapperProps) {
  const setFocusedNodeId = useCanvasStore(state => state.setFocusedNodeId);

  const handleClick = (e: React.MouseEvent) => {
    const target = e.target as HTMLElement;

    // Check for interactive elements that should stop propagation
    const interactiveSelectors = [
      'button',
      'input',
      'span',
      'textarea',
      'select',
    ];

    const isInteractiveElement = interactiveSelectors.some(selector =>
      target.tagName.toLowerCase() === selector || target.onclick !== null
    );

    if (isInteractiveElement) {
      e.stopPropagation();

      // Set focused node after a brief delay to ensure the click event is processed
      if (nodeId) {
        setTimeout(() => {
          setFocusedNodeId(nodeId);
        }, 0);
      }
    }
  };

  return (
    <div
      className={twMerge('w-full h-full text-left', className)}
      onClick={handleClick}
    >
      {children}
    </div>
  );
}