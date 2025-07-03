import { ArgsStoryFn, RenderContext } from '@storybook/types';
import { R as ReactRenderer } from './types-0fc72a6d.js';
import 'react';

declare const render: ArgsStoryFn<ReactRenderer>;
declare function renderToCanvas({ storyContext, unboundStoryFn, showMain, showException, forceRemount, }: RenderContext<ReactRenderer>, canvasElement: ReactRenderer['canvasElement']): Promise<() => void>;

declare const parameters: {};

export { parameters, render, renderToCanvas };
