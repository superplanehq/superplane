export * from '@storybook/blocks';
import { Renderer, DocsRenderFunction } from '@storybook/types';

declare class DocsRenderer<TRenderer extends Renderer> {
    render: DocsRenderFunction<TRenderer>;
    unmount: (element: HTMLElement) => void;
    constructor();
}

export { DocsRenderer };
