import { ReactElement } from 'react';

declare const renderElement: (node: ReactElement, el: Element) => Promise<unknown>;
declare const unmountElement: (el: Element, shouldUseNewRootApi?: boolean) => void;

export { renderElement, unmountElement };
