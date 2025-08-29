/**
 * Stage Update Queue System
 * 
 * Prevents race conditions between syncStageEvents and pollStageUntilNoPending
 * by queuing operations per stage ID and executing them serially.
 */

type StageUpdateOperation = {
  stageId: string;
  operation: () => Promise<void>;
  priority: 'high' | 'normal';
};

class StageUpdateQueue {
  private queues: Map<string, StageUpdateOperation[]> = new Map();
  private processing: Set<string> = new Set();

  /**
   * Add an operation to the queue for a specific stage
   */
  async enqueue(stageId: string, operation: () => Promise<void>, priority: 'high' | 'normal' = 'normal'): Promise<void> {
    const op: StageUpdateOperation = { stageId, operation, priority };
    
    if (!this.queues.has(stageId)) {
      this.queues.set(stageId, []);
    }
    
    const queue = this.queues.get(stageId)!;
    
    if (priority === 'high') {
      // Insert high priority operations at the front
      queue.unshift(op);
    } else {
      queue.push(op);
    }
    
    // Start processing if not already running
    if (!this.processing.has(stageId)) {
      this.processQueue(stageId);
    }
  }

  private async processQueue(stageId: string): Promise<void> {
    if (this.processing.has(stageId)) {
      return;
    }

    this.processing.add(stageId);
    const queue = this.queues.get(stageId);
    
    if (!queue) {
      this.processing.delete(stageId);
      return;
    }

    while (queue.length > 0) {
      const operation = queue.shift()!;
      
      try {
        await operation.operation();
      } catch (error) {
        console.error(`Stage update operation failed for stage ${stageId}:`, error);
      }
    }
    
    this.processing.delete(stageId);
    this.queues.delete(stageId);
  }

  /**
   * Check if a stage has pending operations
   */
  hasPendingOperations(stageId: string): boolean {
    const queue = this.queues.get(stageId);
    return (queue && queue.length > 0) || this.processing.has(stageId);
  }

  /**
   * Get the number of pending operations for a stage
   */
  getPendingCount(stageId: string): number {
    const queue = this.queues.get(stageId);
    return queue ? queue.length : 0;
  }
}

// Global singleton instance
export const stageUpdateQueue = new StageUpdateQueue();