import type { KeyCode } from '../types';
export interface UseKeyPressOptions {
    target?: Window | Document | HTMLElement | ShadowRoot | null;
    actInsideInputWithModifier?: boolean;
}
declare const _default: (keyCode?: KeyCode | null, options?: UseKeyPressOptions) => boolean;
export default _default;
//# sourceMappingURL=useKeyPress.d.ts.map