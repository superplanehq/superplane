import React from 'react';
import { Handle } from 'reactflow';
import Tippy from '@tippyjs/react';
import 'tippy.js/dist/tippy.css';
import 'tippy.js/themes/light.css';
import 'tippy.js/themes/light-border.css';

const BAR_WIDTH = 48;
const BAR_HEIGHT = 6;

const actionsDemo = ['Development Environment: new_run'];
const conditionsDemo = ['run.state=pass', 'output.type=community'];

function TooltipContent() {
  return (
    <div className="p-2 min-w-[300px]">
      <div className="text-xs text-gray-600 font-semibold mb-1">Events this stage listens:</div>
      <div className="flex gap-1 mb-2 flex-wrap">
        {actionsDemo.map((action, i) => (
          <span
            key={action}
            className="bg-indigo-100 text-indigo-800 text-xs font-semibold px-2 py-0.5 rounded mr-1 mb-1 border border-indigo-200"
          >
            {action}
          </span>
        ))}
      </div>
      <div className="text-xs text-gray-600 font-semibold mb-1">Conditions:</div>
      <div className="flex gap-1 mb-2 flex-wrap">
        {conditionsDemo.map((cond, i) => (
          <span
            key={cond}
            className="bg-green-100 text-green-800 text-xs font-semibold px-2 py-0.5 rounded mr-1 mb-1 border border-green-200"
          >
            {cond}
          </span>
        ))}
      </div>
      <div className="bg-white border border-gray-200 rounded p-2 text-xs text-gray-700 shadow-sm">
        Lorem ipsum dolor sit amet, consectetur adipiscing elit. Suspendisse et urna fringilla, tincidunt nulla nec, dictum erat.
      </div>
    </div>
  );
}

export default function CustomBarHandle({ type, position, id }) {
  // Positioning for left/right bars
  const isLeft = position === 'left';
  const isRight = position === 'right';
  const isVertical = isLeft || isRight;
  let placement = 'top';
  if (isLeft) placement = 'left-start';
  if (isRight) placement = 'right-start';

  // --- Fix: Use getReferenceClientRect for zoom-stable positioning ---
  const handleRef = React.useRef(null);
  return (
    <Tippy
      content={<TooltipContent />}
      interactive={true}
      placement={placement}
      delay={[120, 50]}
      theme="light-border"
      maxWidth={320}
      arrow={true}
      offset={[0, 8]}
      getReferenceClientRect={() => {
        if (handleRef.current) {
          return handleRef.current.getBoundingClientRect();
        }
        return undefined;
      }}
    >
      <div style={{ display: 'inline-block' }} ref={handleRef}>
        <Handle
          type={type}
          position={position}
          id={id}
          style={{
            background: 'var(--indigo)',
            borderRadius: 3,
            width: isVertical ? BAR_HEIGHT : BAR_WIDTH,
            height: isVertical ? BAR_WIDTH : BAR_HEIGHT,
            border: 'none',
            left: isLeft ? -BAR_HEIGHT / 2 : undefined,
            right: isRight ? -BAR_HEIGHT / 2 : undefined,
            top: '50%',
            transform: 'translateY(-50%)',
            zIndex: 2,
            boxShadow: '0 1px 6px 0 rgba(19,198,179,0.15)',
          }}
          className="custom-bar-handle"
        />
      </div>
    </Tippy>
  );
}
