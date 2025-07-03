import { ComponentType, ReactElement } from 'react';
import { WebRenderer } from '@storybook/types';

interface ReactRenderer extends WebRenderer {
    component: ComponentType<this['T']>;
    storyResult: StoryFnReactReturnType;
}
type StoryFnReactReturnType = ReactElement<unknown>;

export { ReactRenderer as R };
