import { makeUseVisualState } from '../../motion/utils/use-visual-state.mjs';
import { createSvgRenderState } from './utils/create-render-state.mjs';
import { scrapeMotionValuesFromProps } from './utils/scrape-motion-values.mjs';

const svgMotionConfig = {
    useVisualState: makeUseVisualState({
        scrapeMotionValuesFromProps: scrapeMotionValuesFromProps,
        createRenderState: createSvgRenderState,
    }),
};

export { svgMotionConfig };
