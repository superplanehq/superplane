import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { HistoryTab } from './HistoryTab';
import { Stage } from '../../store/types';

vi.mock('@/hooks/useCanvasData');

const mockStage: Stage = {
  metadata: {
    id: 'stage-1',
    name: 'Test Stage',
  },
  queue: [],
  executions: [],
};

const mockUseStageQueueEvents = {
  data: { pages: [] },
  fetchNextPage: vi.fn(),
  hasNextPage: false,
  isFetchingNextPage: false,
  refetch: vi.fn(),
};

const mockUseStageEvents = {
  data: { pages: [] },
  fetchNextPage: vi.fn(),
  hasNextPage: false,
  isFetchingNextPage: false,
  refetch: vi.fn(),
};

vi.mock('@/hooks/useCanvasData', () => ({
  useOrganizationUsersForCanvas: vi.fn(() => ({ data: [] })),
  useStageQueueEvents: vi.fn(() => mockUseStageQueueEvents),
  useStageEvents: vi.fn(() => mockUseStageEvents),
}));

const renderWithQueryClient = (component: React.ReactElement) => {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } }
  });
  return render(
    <QueryClientProvider client={queryClient}>
      {component}
    </QueryClientProvider>
  );
};

describe('HistoryTab', () => {
  it('renders history tab with empty state', () => {
    renderWithQueryClient(
      <HistoryTab
        selectedStage={mockStage}
        organizationId="org-1"
        canvasId="canvas-1"
        approveStageEvent={vi.fn()}
        discardStageEvent={vi.fn()}
      />
    );

    expect(screen.getByText('History (0 items)')).toBeInTheDocument();
    expect(screen.getByText('No history available')).toBeInTheDocument();
  });

  it('renders search input and filter tabs', () => {
    renderWithQueryClient(
      <HistoryTab
        selectedStage={mockStage}
        organizationId="org-1"
        canvasId="canvas-1"
        approveStageEvent={vi.fn()}
        discardStageEvent={vi.fn()}
      />
    );

    expect(screen.getByPlaceholderText('Search history...')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'All' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Runs' })).toBeInTheDocument();
  });
});