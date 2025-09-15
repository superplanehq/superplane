import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';

vi.mock('./store/canvasStore');
vi.mock('./hooks/useWebsocketEvents');
vi.mock('./hooks/useAutoLayout');
vi.mock('./utils/nodeHandlers');
vi.mock('@/api-client');
vi.mock('./components/FlowRenderer', () => ({
  FlowRenderer: () => <div>FlowRenderer Mock</div>
}));
vi.mock('@/canvas/store/canvasStore', () => ({
  useCanvasStore: vi.fn(() => ({
    initialize: vi.fn(),
    selectedStageId: null,
    cleanSelectedStageId: vi.fn(),
    selectedEventSourceId: null,
    cleanSelectedEventSourceId: vi.fn(),
    editingStageId: null,
    stages: [],
    eventSources: [],
    connectionGroups: [],
    approveStageEvent: vi.fn(),
    discardStageEvent: vi.fn(),
    cancelStageExecution: vi.fn(),
    fitViewNode: vi.fn(),
    lockedNodes: false,
    setFocusedNodeId: vi.fn(),
    setNodes: vi.fn(),
  }))
}));

const mockUseCanvasStore = {
  initialize: vi.fn(),
  selectedStageId: null,
  cleanSelectedStageId: vi.fn(),
  selectedEventSourceId: null,
  cleanSelectedEventSourceId: vi.fn(),
  editingStageId: null,
  stages: [],
  eventSources: [],
  connectionGroups: [],
  approveStageEvent: vi.fn(),
  discardStageEvent: vi.fn(),
  cancelStageExecution: vi.fn(),
  fitViewNode: vi.fn(),
  lockedNodes: false,
  setFocusedNodeId: vi.fn(),
  setNodes: vi.fn(),
};

vi.mock('./store/canvasStore', () => ({
  useCanvasStore: vi.fn(() => mockUseCanvasStore),
}));

vi.mock('./hooks/useWebsocketEvents', () => ({
  useWebsocketEvents: vi.fn(),
}));

vi.mock('./utils/nodeHandlers', () => ({
  useNodeHandlers: vi.fn(() => ({ handleAddNode: vi.fn() })),
}));

vi.mock('./hooks/useAutoLayout', () => ({
  useAutoLayout: vi.fn(() => ({ applyElkAutoLayout: vi.fn() })),
}));

vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom');
  return {
    ...actual,
    useParams: vi.fn(() => ({ organizationId: 'org123', canvasId: 'canvas123' })),
    useLocation: vi.fn(() => ({ pathname: '/canvas/123', hash: '' })),
    useNavigate: vi.fn(() => vi.fn()),
  };
});

const { Canvas } = await import('./index');

const renderWithRouter = (component: React.ReactElement) => {
  return render(
    <BrowserRouter>
      {component}
    </BrowserRouter>
  );
};

describe('Canvas', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders loading state initially', () => {
    renderWithRouter(<Canvas />);
    expect(screen.getByText('Loading canvas...')).toBeInTheDocument();
  });

  it('renders error state when canvas ID is missing', async () => {
    const { useParams } = await import('react-router-dom');
    vi.mocked(useParams).mockReturnValue({
      organizationId: 'org123',
      canvasId: undefined,
    });
    
    renderWithRouter(<Canvas />);
    expect(screen.getByText(/No canvas ID provided/)).toBeInTheDocument();
  });
});