import { type HTMLAttributes, type ReactNode } from 'react';
import type { PanelPosition } from '../../types';
export type PanelProps = HTMLAttributes<HTMLDivElement> & {
    position: PanelPosition;
    children: ReactNode;
};
declare function Panel({ position, children, className, style, ...rest }: PanelProps): JSX.Element;
export default Panel;
//# sourceMappingURL=index.d.ts.map