import { createUnplugin } from 'unplugin';
import fs from 'fs/promises';
import { loadCsf, enrichCsf, formatCsf } from '@storybook/csf-tools';

var STORIES_REGEX=/\.(story|stories)\.[tj]sx?$/,logger=console,unplugin=createUnplugin(options=>({name:"unplugin-csf",transformInclude(id){return STORIES_REGEX.test(id)},async transform(code,id){let sourceCode=await fs.readFile(id,"utf-8");try{let makeTitle=userTitle=>userTitle||"default",csf=loadCsf(code,{makeTitle}).parse(),csfSource=loadCsf(sourceCode,{makeTitle}).parse();return enrichCsf(csf,csfSource,options),formatCsf(csf,{sourceMaps:!0})}catch(err){return err.message?.startsWith("CSF:")||logger.warn(err.message),code}}})),{esbuild}=unplugin,{webpack}=unplugin,{rollup}=unplugin,{vite}=unplugin;

export { esbuild, rollup, unplugin, vite, webpack };
