import {useLayoutEffect as $f0a04ccd8dbdd83b$export$e5c5a5f917a5871c} from "./useLayoutEffect.mjs";
import {useRef as $azsE2$useRef} from "react";

/*
 * Copyright 2024 Adobe. All rights reserved.
 * This file is licensed to you under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License. You may obtain a copy
 * of the License at http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software distributed under
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR REPRESENTATIONS
 * OF ANY KIND, either express or implied. See the License for the specific language
 * governing permissions and limitations under the License.
 */ 

function $ca9b37712f007381$export$72ef708ab07251f1(effect, dependencies) {
    const isInitialMount = (0, $azsE2$useRef)(true);
    const lastDeps = (0, $azsE2$useRef)(null);
    (0, $f0a04ccd8dbdd83b$export$e5c5a5f917a5871c)(()=>{
        isInitialMount.current = true;
        return ()=>{
            isInitialMount.current = false;
        };
    }, []);
    (0, $f0a04ccd8dbdd83b$export$e5c5a5f917a5871c)(()=>{
        if (isInitialMount.current) isInitialMount.current = false;
        else if (!lastDeps.current || dependencies.some((dep, i)=>!Object.is(dep, lastDeps[i]))) effect();
        lastDeps.current = dependencies;
    // eslint-disable-next-line react-hooks/exhaustive-deps
    }, dependencies);
}


export {$ca9b37712f007381$export$72ef708ab07251f1 as useUpdateLayoutEffect};
//# sourceMappingURL=useUpdateLayoutEffect.module.js.map
