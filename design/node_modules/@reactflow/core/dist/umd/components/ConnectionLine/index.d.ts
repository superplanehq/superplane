import { type CSSProperties } from 'react';
import type { ConnectionLineComponent } from '../../types';
import { ConnectionLineType } from '../../types';
type ConnectionLineWrapperProps = {
    type: ConnectionLineType;
    component?: ConnectionLineComponent;
    containerStyle?: CSSProperties;
    style?: CSSProperties;
};
declare function ConnectionLineWrapper({ containerStyle, style, type, component }: ConnectionLineWrapperProps): JSX.Element | null;
export default ConnectionLineWrapper;
//# sourceMappingURL=index.d.ts.map