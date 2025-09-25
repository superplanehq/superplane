import type { CanvasCardData } from '../../src/components/CanvasCard'

export const mockCanvas: CanvasCardData = {
  id: '123e4567-e89b-12d3-a456-426614174000',
  name: 'Product Analytics Dashboard',
  description: 'A comprehensive dashboard for tracking product metrics, user engagement, and conversion rates across multiple platforms.',
  createdAt: '2024-01-15',
  createdBy: {
    name: 'John Smith',
    initials: 'JS',
  },
  type: 'canvas' as const,
}

export const shortCanvas: CanvasCardData = {
  id: '456e7890-e89b-12d3-a456-426614174001',
  name: 'API Status',
  description: 'Simple API monitoring.',
  createdAt: '2024-01-20',
  createdBy: {
    name: 'Jane Doe',
    initials: 'JD',
  },
  type: 'canvas' as const,
}

export const noDescriptionCanvas: CanvasCardData = {
  id: '789e0123-e89b-12d3-a456-426614174002',
  name: 'Untitled Canvas',
  createdAt: '2024-01-25',
  createdBy: {
    name: 'Alex Johnson',
    initials: 'AJ',
  },
  type: 'canvas' as const,
}

export const canvasMocks = {
  mockCanvas,
  shortCanvas,
  noDescriptionCanvas,
}