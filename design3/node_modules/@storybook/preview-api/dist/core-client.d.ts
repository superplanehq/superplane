import { C as ClientApi } from './ClientApi-24d2bfd0.js';
export { S as StoryStore } from './StoryStore-f7424ddf.js';
import { Renderer, ProjectAnnotations, ArgsStoryFn } from '@storybook/types';
import 'synchronous-promise';
import './hooks-655fa363.js';

interface CoreClient_RendererImplementation<TRenderer extends Renderer> {
    /**
     * A function that applies decorators to a story.
     * @template TRenderer The type of renderer used by the Storybook client API.
     * @type {ProjectAnnotations<TRenderer>['applyDecorators']}
     */
    decorateStory?: ProjectAnnotations<TRenderer>['applyDecorators'];
    /**
     * A function that renders a story with args.
     * @template TRenderer The type of renderer used by the Storybook client API.
     * @type {ArgsStoryFn<TRenderer>}
     */
    render?: ArgsStoryFn<TRenderer>;
}
interface CoreClient_ClientAPIFacade {
    /**
     * The old way of adding stories at runtime.
     * @deprecated This method is deprecated and will be removed in a future version.
     */
    storiesOf: (...args: any[]) => never;
    /**
     * The old way of retrieving the list of stories at runtime.
     * @deprecated This method is deprecated and will be removed in a future version.
     */
    raw: (...args: any[]) => never;
}
interface CoreClient_StartReturnValue<TRenderer extends Renderer> {
    /**
     * Forces a re-render of all stories in the Storybook preview.
     * This function emits the `FORCE_RE_RENDER` event to the Storybook channel.
     * @deprecated This method is deprecated and will be removed in a future version.
     * @returns {void}
     */
    forceReRender: () => void;
    /**
     * The old way of setting up storybook with runtime configuration.
     * @deprecated This method is deprecated and will be removed in a future version.
     * @returns {void}
     */
    configure: any;
    /**
     * @deprecated This property is deprecated and will be removed in a future version.
     * @type {ClientApi<TRenderer> | CoreClient_ClientAPIFacade}
     */
    clientApi: ClientApi<TRenderer> | CoreClient_ClientAPIFacade;
}
/**
 * Initializes the Storybook preview API.
 * @template TRenderer The type of renderer used by the Storybook client API.
 * @param {ProjectAnnotations<TRenderer>['renderToCanvas']} renderToCanvas A function that renders a story to a canvas.
 * @param {CoreClient_RendererImplementation<TRenderer>} [options] Optional configuration options for the renderer implementation.
 * @param {ProjectAnnotations<TRenderer>['applyDecorators']} [options.decorateStory] A function that applies decorators to a story.
 * @param {ArgsStoryFn<TRenderer>} [options.render] A function that renders a story with arguments.
 * @returns {CoreClient_StartReturnValue<TRenderer>} An object containing functions and objects related to the Storybook preview API.
 */
declare function start<TRenderer extends Renderer>(renderToCanvas: ProjectAnnotations<TRenderer>['renderToCanvas'], { decorateStory, render }?: CoreClient_RendererImplementation<TRenderer>): CoreClient_StartReturnValue<TRenderer>;

export { ClientApi, start };
