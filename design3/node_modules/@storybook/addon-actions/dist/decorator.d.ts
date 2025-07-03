import { Renderer, PartialStoryFn } from '@storybook/types';

declare const withActions: <T extends Renderer>(storyFn: PartialStoryFn<T>) => T['storyResult'];

export { withActions };
