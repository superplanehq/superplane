import {version as $iulvE$version} from "react";


function $cdc5a6778b766db2$export$a9d04c5684123369(value) {
    const pieces = (0, $iulvE$version).split('.');
    const major = parseInt(pieces[0], 10);
    if (major >= 19) return value;
    // compatibility with React < 19
    return value ? 'true' : undefined;
}


export {$cdc5a6778b766db2$export$a9d04c5684123369 as inertValue};
//# sourceMappingURL=inertValue.module.js.map
