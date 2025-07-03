import { global } from '@storybook/global';
import { logger } from '@storybook/client-logger';

var __create=Object.create;var __defProp=Object.defineProperty;var __getOwnPropDesc=Object.getOwnPropertyDescriptor;var __getOwnPropNames=Object.getOwnPropertyNames;var __getProtoOf=Object.getPrototypeOf,__hasOwnProp=Object.prototype.hasOwnProperty;var __commonJS=(cb,mod)=>function(){return mod||(0, cb[__getOwnPropNames(cb)[0]])((mod={exports:{}}).exports,mod),mod.exports};var __copyProps=(to,from,except,desc)=>{if(from&&typeof from=="object"||typeof from=="function")for(let key of __getOwnPropNames(from))!__hasOwnProp.call(to,key)&&key!==except&&__defProp(to,key,{get:()=>from[key],enumerable:!(desc=__getOwnPropDesc(from,key))||desc.enumerable});return to};var __toESM=(mod,isNodeMode,target)=>(target=mod!=null?__create(__getProtoOf(mod)):{},__copyProps(isNodeMode||!mod||!mod.__esModule?__defProp(target,"default",{value:mod,enumerable:!0}):target,mod));function _extends(){return _extends=Object.assign?Object.assign.bind():function(target){for(var i=1;i<arguments.length;i++){var source=arguments[i];for(var key in source)Object.prototype.hasOwnProperty.call(source,key)&&(target[key]=source[key]);}return target},_extends.apply(this,arguments)}function _assertThisInitialized(self){if(self===void 0)throw new ReferenceError("this hasn't been initialised - super() hasn't been called");return self}function _setPrototypeOf(o,p){return _setPrototypeOf=Object.setPrototypeOf?Object.setPrototypeOf.bind():function(o2,p2){return o2.__proto__=p2,o2},_setPrototypeOf(o,p)}function _inheritsLoose(subClass,superClass){subClass.prototype=Object.create(superClass.prototype),subClass.prototype.constructor=subClass,_setPrototypeOf(subClass,superClass);}function _getPrototypeOf(o){return _getPrototypeOf=Object.setPrototypeOf?Object.getPrototypeOf.bind():function(o2){return o2.__proto__||Object.getPrototypeOf(o2)},_getPrototypeOf(o)}function _isNativeFunction(fn){try{return Function.toString.call(fn).indexOf("[native code]")!==-1}catch{return typeof fn=="function"}}function _isNativeReflectConstruct(){try{var t=!Boolean.prototype.valueOf.call(Reflect.construct(Boolean,[],function(){}));}catch{}return (_isNativeReflectConstruct=function(){return !!t})()}function _construct(t,e,r){if(_isNativeReflectConstruct())return Reflect.construct.apply(null,arguments);var o=[null];o.push.apply(o,e);var p=new(t.bind.apply(t,o));return r&&_setPrototypeOf(p,r.prototype),p}function _wrapNativeSuper(Class){var _cache=typeof Map=="function"?new Map:void 0;return _wrapNativeSuper=function(Class2){if(Class2===null||!_isNativeFunction(Class2))return Class2;if(typeof Class2!="function")throw new TypeError("Super expression must either be null or a function");if(typeof _cache<"u"){if(_cache.has(Class2))return _cache.get(Class2);_cache.set(Class2,Wrapper);}function Wrapper(){return _construct(Class2,arguments,_getPrototypeOf(this).constructor)}return Wrapper.prototype=Object.create(Class2.prototype,{constructor:{value:Wrapper,enumerable:!1,writable:!0,configurable:!0}}),_setPrototypeOf(Wrapper,Class2)},_wrapNativeSuper(Class)}var ERRORS={1:`Passed invalid arguments to hsl, please pass multiple numbers e.g. hsl(360, 0.75, 0.4) or an object e.g. rgb({ hue: 255, saturation: 0.4, lightness: 0.75 }).

`,2:`Passed invalid arguments to hsla, please pass multiple numbers e.g. hsla(360, 0.75, 0.4, 0.7) or an object e.g. rgb({ hue: 255, saturation: 0.4, lightness: 0.75, alpha: 0.7 }).

`,3:`Passed an incorrect argument to a color function, please pass a string representation of a color.

`,4:`Couldn't generate valid rgb string from %s, it returned %s.

`,5:`Couldn't parse the color string. Please provide the color as a string in hex, rgb, rgba, hsl or hsla notation.

`,6:`Passed invalid arguments to rgb, please pass multiple numbers e.g. rgb(255, 205, 100) or an object e.g. rgb({ red: 255, green: 205, blue: 100 }).

`,7:`Passed invalid arguments to rgba, please pass multiple numbers e.g. rgb(255, 205, 100, 0.75) or an object e.g. rgb({ red: 255, green: 205, blue: 100, alpha: 0.75 }).

`,8:`Passed invalid argument to toColorString, please pass a RgbColor, RgbaColor, HslColor or HslaColor object.

`,9:`Please provide a number of steps to the modularScale helper.

`,10:`Please pass a number or one of the predefined scales to the modularScale helper as the ratio.

`,11:`Invalid value passed as base to modularScale, expected number or em string but got "%s"

`,12:`Expected a string ending in "px" or a number passed as the first argument to %s(), got "%s" instead.

`,13:`Expected a string ending in "px" or a number passed as the second argument to %s(), got "%s" instead.

`,14:`Passed invalid pixel value ("%s") to %s(), please pass a value like "12px" or 12.

`,15:`Passed invalid base value ("%s") to %s(), please pass a value like "12px" or 12.

`,16:`You must provide a template to this method.

`,17:`You passed an unsupported selector state to this method.

`,18:`minScreen and maxScreen must be provided as stringified numbers with the same units.

`,19:`fromSize and toSize must be provided as stringified numbers with the same units.

`,20:`expects either an array of objects or a single object with the properties prop, fromSize, and toSize.

`,21:"expects the objects in the first argument array to have the properties `prop`, `fromSize`, and `toSize`.\n\n",22:"expects the first argument object to have the properties `prop`, `fromSize`, and `toSize`.\n\n",23:`fontFace expects a name of a font-family.

`,24:`fontFace expects either the path to the font file(s) or a name of a local copy.

`,25:`fontFace expects localFonts to be an array.

`,26:`fontFace expects fileFormats to be an array.

`,27:`radialGradient requries at least 2 color-stops to properly render.

`,28:`Please supply a filename to retinaImage() as the first argument.

`,29:`Passed invalid argument to triangle, please pass correct pointingDirection e.g. 'right'.

`,30:"Passed an invalid value to `height` or `width`. Please provide a pixel based unit.\n\n",31:`The animation shorthand only takes 8 arguments. See the specification for more information: http://mdn.io/animation

`,32:`To pass multiple animations please supply them in arrays, e.g. animation(['rotate', '2s'], ['move', '1s'])
To pass a single animation please supply them in simple values, e.g. animation('rotate', '2s')

`,33:`The animation shorthand arrays can only have 8 elements. See the specification for more information: http://mdn.io/animation

`,34:`borderRadius expects a radius value as a string or number as the second argument.

`,35:`borderRadius expects one of "top", "bottom", "left" or "right" as the first argument.

`,36:`Property must be a string value.

`,37:`Syntax Error at %s.

`,38:`Formula contains a function that needs parentheses at %s.

`,39:`Formula is missing closing parenthesis at %s.

`,40:`Formula has too many closing parentheses at %s.

`,41:`All values in a formula must have the same unit or be unitless.

`,42:`Please provide a number of steps to the modularScale helper.

`,43:`Please pass a number or one of the predefined scales to the modularScale helper as the ratio.

`,44:`Invalid value passed as base to modularScale, expected number or em/rem string but got %s.

`,45:`Passed invalid argument to hslToColorString, please pass a HslColor or HslaColor object.

`,46:`Passed invalid argument to rgbToColorString, please pass a RgbColor or RgbaColor object.

`,47:`minScreen and maxScreen must be provided as stringified numbers with the same units.

`,48:`fromSize and toSize must be provided as stringified numbers with the same units.

`,49:`Expects either an array of objects or a single object with the properties prop, fromSize, and toSize.

`,50:`Expects the objects in the first argument array to have the properties prop, fromSize, and toSize.

`,51:`Expects the first argument object to have the properties prop, fromSize, and toSize.

`,52:`fontFace expects either the path to the font file(s) or a name of a local copy.

`,53:`fontFace expects localFonts to be an array.

`,54:`fontFace expects fileFormats to be an array.

`,55:`fontFace expects a name of a font-family.

`,56:`linearGradient requries at least 2 color-stops to properly render.

`,57:`radialGradient requries at least 2 color-stops to properly render.

`,58:`Please supply a filename to retinaImage() as the first argument.

`,59:`Passed invalid argument to triangle, please pass correct pointingDirection e.g. 'right'.

`,60:"Passed an invalid value to `height` or `width`. Please provide a pixel based unit.\n\n",61:`Property must be a string value.

`,62:`borderRadius expects a radius value as a string or number as the second argument.

`,63:`borderRadius expects one of "top", "bottom", "left" or "right" as the first argument.

`,64:`The animation shorthand only takes 8 arguments. See the specification for more information: http://mdn.io/animation.

`,65:`To pass multiple animations please supply them in arrays, e.g. animation(['rotate', '2s'], ['move', '1s'])\\nTo pass a single animation please supply them in simple values, e.g. animation('rotate', '2s').

`,66:`The animation shorthand arrays can only have 8 elements. See the specification for more information: http://mdn.io/animation.

`,67:`You must provide a template to this method.

`,68:`You passed an unsupported selector state to this method.

`,69:`Expected a string ending in "px" or a number passed as the first argument to %s(), got %s instead.

`,70:`Expected a string ending in "px" or a number passed as the second argument to %s(), got %s instead.

`,71:`Passed invalid pixel value %s to %s(), please pass a value like "12px" or 12.

`,72:`Passed invalid base value %s to %s(), please pass a value like "12px" or 12.

`,73:`Please provide a valid CSS variable.

`,74:`CSS variable not found and no default was provided.

`,75:`important requires a valid style object, got a %s instead.

`,76:`fromSize and toSize must be provided as stringified numbers with the same units as minScreen and maxScreen.

`,77:`remToPx expects a value in "rem" but you provided it in "%s".

`,78:`base must be set in "px" or "%" but you set it in "%s".
`};function format(){for(var _len=arguments.length,args=new Array(_len),_key=0;_key<_len;_key++)args[_key]=arguments[_key];var a=args[0],b=[],c;for(c=1;c<args.length;c+=1)b.push(args[c]);return b.forEach(function(d){a=a.replace(/%[a-z]/,d);}),a}var PolishedError=function(_Error){_inheritsLoose(PolishedError2,_Error);function PolishedError2(code){for(var _this,_len2=arguments.length,args=new Array(_len2>1?_len2-1:0),_key2=1;_key2<_len2;_key2++)args[_key2-1]=arguments[_key2];return _this=_Error.call(this,format.apply(void 0,[ERRORS[code]].concat(args)))||this,_assertThisInitialized(_this)}return PolishedError2}(_wrapNativeSuper(Error));function colorToInt(color2){return Math.round(color2*255)}function convertToInt(red,green,blue){return colorToInt(red)+","+colorToInt(green)+","+colorToInt(blue)}function hslToRgb(hue,saturation,lightness,convert){if(convert===void 0&&(convert=convertToInt),saturation===0)return convert(lightness,lightness,lightness);var huePrime=(hue%360+360)%360/60,chroma=(1-Math.abs(2*lightness-1))*saturation,secondComponent=chroma*(1-Math.abs(huePrime%2-1)),red=0,green=0,blue=0;huePrime>=0&&huePrime<1?(red=chroma,green=secondComponent):huePrime>=1&&huePrime<2?(red=secondComponent,green=chroma):huePrime>=2&&huePrime<3?(green=chroma,blue=secondComponent):huePrime>=3&&huePrime<4?(green=secondComponent,blue=chroma):huePrime>=4&&huePrime<5?(red=secondComponent,blue=chroma):huePrime>=5&&huePrime<6&&(red=chroma,blue=secondComponent);var lightnessModification=lightness-chroma/2,finalRed=red+lightnessModification,finalGreen=green+lightnessModification,finalBlue=blue+lightnessModification;return convert(finalRed,finalGreen,finalBlue)}var namedColorMap={aliceblue:"f0f8ff",antiquewhite:"faebd7",aqua:"00ffff",aquamarine:"7fffd4",azure:"f0ffff",beige:"f5f5dc",bisque:"ffe4c4",black:"000",blanchedalmond:"ffebcd",blue:"0000ff",blueviolet:"8a2be2",brown:"a52a2a",burlywood:"deb887",cadetblue:"5f9ea0",chartreuse:"7fff00",chocolate:"d2691e",coral:"ff7f50",cornflowerblue:"6495ed",cornsilk:"fff8dc",crimson:"dc143c",cyan:"00ffff",darkblue:"00008b",darkcyan:"008b8b",darkgoldenrod:"b8860b",darkgray:"a9a9a9",darkgreen:"006400",darkgrey:"a9a9a9",darkkhaki:"bdb76b",darkmagenta:"8b008b",darkolivegreen:"556b2f",darkorange:"ff8c00",darkorchid:"9932cc",darkred:"8b0000",darksalmon:"e9967a",darkseagreen:"8fbc8f",darkslateblue:"483d8b",darkslategray:"2f4f4f",darkslategrey:"2f4f4f",darkturquoise:"00ced1",darkviolet:"9400d3",deeppink:"ff1493",deepskyblue:"00bfff",dimgray:"696969",dimgrey:"696969",dodgerblue:"1e90ff",firebrick:"b22222",floralwhite:"fffaf0",forestgreen:"228b22",fuchsia:"ff00ff",gainsboro:"dcdcdc",ghostwhite:"f8f8ff",gold:"ffd700",goldenrod:"daa520",gray:"808080",green:"008000",greenyellow:"adff2f",grey:"808080",honeydew:"f0fff0",hotpink:"ff69b4",indianred:"cd5c5c",indigo:"4b0082",ivory:"fffff0",khaki:"f0e68c",lavender:"e6e6fa",lavenderblush:"fff0f5",lawngreen:"7cfc00",lemonchiffon:"fffacd",lightblue:"add8e6",lightcoral:"f08080",lightcyan:"e0ffff",lightgoldenrodyellow:"fafad2",lightgray:"d3d3d3",lightgreen:"90ee90",lightgrey:"d3d3d3",lightpink:"ffb6c1",lightsalmon:"ffa07a",lightseagreen:"20b2aa",lightskyblue:"87cefa",lightslategray:"789",lightslategrey:"789",lightsteelblue:"b0c4de",lightyellow:"ffffe0",lime:"0f0",limegreen:"32cd32",linen:"faf0e6",magenta:"f0f",maroon:"800000",mediumaquamarine:"66cdaa",mediumblue:"0000cd",mediumorchid:"ba55d3",mediumpurple:"9370db",mediumseagreen:"3cb371",mediumslateblue:"7b68ee",mediumspringgreen:"00fa9a",mediumturquoise:"48d1cc",mediumvioletred:"c71585",midnightblue:"191970",mintcream:"f5fffa",mistyrose:"ffe4e1",moccasin:"ffe4b5",navajowhite:"ffdead",navy:"000080",oldlace:"fdf5e6",olive:"808000",olivedrab:"6b8e23",orange:"ffa500",orangered:"ff4500",orchid:"da70d6",palegoldenrod:"eee8aa",palegreen:"98fb98",paleturquoise:"afeeee",palevioletred:"db7093",papayawhip:"ffefd5",peachpuff:"ffdab9",peru:"cd853f",pink:"ffc0cb",plum:"dda0dd",powderblue:"b0e0e6",purple:"800080",rebeccapurple:"639",red:"f00",rosybrown:"bc8f8f",royalblue:"4169e1",saddlebrown:"8b4513",salmon:"fa8072",sandybrown:"f4a460",seagreen:"2e8b57",seashell:"fff5ee",sienna:"a0522d",silver:"c0c0c0",skyblue:"87ceeb",slateblue:"6a5acd",slategray:"708090",slategrey:"708090",snow:"fffafa",springgreen:"00ff7f",steelblue:"4682b4",tan:"d2b48c",teal:"008080",thistle:"d8bfd8",tomato:"ff6347",turquoise:"40e0d0",violet:"ee82ee",wheat:"f5deb3",white:"fff",whitesmoke:"f5f5f5",yellow:"ff0",yellowgreen:"9acd32"};function nameToHex(color2){if(typeof color2!="string")return color2;var normalizedColorName=color2.toLowerCase();return namedColorMap[normalizedColorName]?"#"+namedColorMap[normalizedColorName]:color2}var hexRegex=/^#[a-fA-F0-9]{6}$/,hexRgbaRegex=/^#[a-fA-F0-9]{8}$/,reducedHexRegex=/^#[a-fA-F0-9]{3}$/,reducedRgbaHexRegex=/^#[a-fA-F0-9]{4}$/,rgbRegex=/^rgb\(\s*(\d{1,3})\s*(?:,)?\s*(\d{1,3})\s*(?:,)?\s*(\d{1,3})\s*\)$/i,rgbaRegex=/^rgb(?:a)?\(\s*(\d{1,3})\s*(?:,)?\s*(\d{1,3})\s*(?:,)?\s*(\d{1,3})\s*(?:,|\/)\s*([-+]?\d*[.]?\d+[%]?)\s*\)$/i,hslRegex=/^hsl\(\s*(\d{0,3}[.]?[0-9]+(?:deg)?)\s*(?:,)?\s*(\d{1,3}[.]?[0-9]?)%\s*(?:,)?\s*(\d{1,3}[.]?[0-9]?)%\s*\)$/i,hslaRegex=/^hsl(?:a)?\(\s*(\d{0,3}[.]?[0-9]+(?:deg)?)\s*(?:,)?\s*(\d{1,3}[.]?[0-9]?)%\s*(?:,)?\s*(\d{1,3}[.]?[0-9]?)%\s*(?:,|\/)\s*([-+]?\d*[.]?\d+[%]?)\s*\)$/i;function parseToRgb(color2){if(typeof color2!="string")throw new PolishedError(3);var normalizedColor=nameToHex(color2);if(normalizedColor.match(hexRegex))return {red:parseInt(""+normalizedColor[1]+normalizedColor[2],16),green:parseInt(""+normalizedColor[3]+normalizedColor[4],16),blue:parseInt(""+normalizedColor[5]+normalizedColor[6],16)};if(normalizedColor.match(hexRgbaRegex)){var alpha=parseFloat((parseInt(""+normalizedColor[7]+normalizedColor[8],16)/255).toFixed(2));return {red:parseInt(""+normalizedColor[1]+normalizedColor[2],16),green:parseInt(""+normalizedColor[3]+normalizedColor[4],16),blue:parseInt(""+normalizedColor[5]+normalizedColor[6],16),alpha}}if(normalizedColor.match(reducedHexRegex))return {red:parseInt(""+normalizedColor[1]+normalizedColor[1],16),green:parseInt(""+normalizedColor[2]+normalizedColor[2],16),blue:parseInt(""+normalizedColor[3]+normalizedColor[3],16)};if(normalizedColor.match(reducedRgbaHexRegex)){var _alpha=parseFloat((parseInt(""+normalizedColor[4]+normalizedColor[4],16)/255).toFixed(2));return {red:parseInt(""+normalizedColor[1]+normalizedColor[1],16),green:parseInt(""+normalizedColor[2]+normalizedColor[2],16),blue:parseInt(""+normalizedColor[3]+normalizedColor[3],16),alpha:_alpha}}var rgbMatched=rgbRegex.exec(normalizedColor);if(rgbMatched)return {red:parseInt(""+rgbMatched[1],10),green:parseInt(""+rgbMatched[2],10),blue:parseInt(""+rgbMatched[3],10)};var rgbaMatched=rgbaRegex.exec(normalizedColor.substring(0,50));if(rgbaMatched)return {red:parseInt(""+rgbaMatched[1],10),green:parseInt(""+rgbaMatched[2],10),blue:parseInt(""+rgbaMatched[3],10),alpha:parseFloat(""+rgbaMatched[4])>1?parseFloat(""+rgbaMatched[4])/100:parseFloat(""+rgbaMatched[4])};var hslMatched=hslRegex.exec(normalizedColor);if(hslMatched){var hue=parseInt(""+hslMatched[1],10),saturation=parseInt(""+hslMatched[2],10)/100,lightness=parseInt(""+hslMatched[3],10)/100,rgbColorString="rgb("+hslToRgb(hue,saturation,lightness)+")",hslRgbMatched=rgbRegex.exec(rgbColorString);if(!hslRgbMatched)throw new PolishedError(4,normalizedColor,rgbColorString);return {red:parseInt(""+hslRgbMatched[1],10),green:parseInt(""+hslRgbMatched[2],10),blue:parseInt(""+hslRgbMatched[3],10)}}var hslaMatched=hslaRegex.exec(normalizedColor.substring(0,50));if(hslaMatched){var _hue=parseInt(""+hslaMatched[1],10),_saturation=parseInt(""+hslaMatched[2],10)/100,_lightness=parseInt(""+hslaMatched[3],10)/100,_rgbColorString="rgb("+hslToRgb(_hue,_saturation,_lightness)+")",_hslRgbMatched=rgbRegex.exec(_rgbColorString);if(!_hslRgbMatched)throw new PolishedError(4,normalizedColor,_rgbColorString);return {red:parseInt(""+_hslRgbMatched[1],10),green:parseInt(""+_hslRgbMatched[2],10),blue:parseInt(""+_hslRgbMatched[3],10),alpha:parseFloat(""+hslaMatched[4])>1?parseFloat(""+hslaMatched[4])/100:parseFloat(""+hslaMatched[4])}}throw new PolishedError(5)}function rgbToHsl(color2){var red=color2.red/255,green=color2.green/255,blue=color2.blue/255,max=Math.max(red,green,blue),min=Math.min(red,green,blue),lightness=(max+min)/2;if(max===min)return color2.alpha!==void 0?{hue:0,saturation:0,lightness,alpha:color2.alpha}:{hue:0,saturation:0,lightness};var hue,delta=max-min,saturation=lightness>.5?delta/(2-max-min):delta/(max+min);switch(max){case red:hue=(green-blue)/delta+(green<blue?6:0);break;case green:hue=(blue-red)/delta+2;break;default:hue=(red-green)/delta+4;break}return hue*=60,color2.alpha!==void 0?{hue,saturation,lightness,alpha:color2.alpha}:{hue,saturation,lightness}}function parseToHsl(color2){return rgbToHsl(parseToRgb(color2))}var reduceHexValue=function(value){return value.length===7&&value[1]===value[2]&&value[3]===value[4]&&value[5]===value[6]?"#"+value[1]+value[3]+value[5]:value},reduceHexValue$1=reduceHexValue;function numberToHex(value){var hex=value.toString(16);return hex.length===1?"0"+hex:hex}function colorToHex(color2){return numberToHex(Math.round(color2*255))}function convertToHex(red,green,blue){return reduceHexValue$1("#"+colorToHex(red)+colorToHex(green)+colorToHex(blue))}function hslToHex(hue,saturation,lightness){return hslToRgb(hue,saturation,lightness,convertToHex)}function hsl(value,saturation,lightness){if(typeof value=="number"&&typeof saturation=="number"&&typeof lightness=="number")return hslToHex(value,saturation,lightness);if(typeof value=="object"&&saturation===void 0&&lightness===void 0)return hslToHex(value.hue,value.saturation,value.lightness);throw new PolishedError(1)}function hsla(value,saturation,lightness,alpha){if(typeof value=="number"&&typeof saturation=="number"&&typeof lightness=="number"&&typeof alpha=="number")return alpha>=1?hslToHex(value,saturation,lightness):"rgba("+hslToRgb(value,saturation,lightness)+","+alpha+")";if(typeof value=="object"&&saturation===void 0&&lightness===void 0&&alpha===void 0)return value.alpha>=1?hslToHex(value.hue,value.saturation,value.lightness):"rgba("+hslToRgb(value.hue,value.saturation,value.lightness)+","+value.alpha+")";throw new PolishedError(2)}function rgb(value,green,blue){if(typeof value=="number"&&typeof green=="number"&&typeof blue=="number")return reduceHexValue$1("#"+numberToHex(value)+numberToHex(green)+numberToHex(blue));if(typeof value=="object"&&green===void 0&&blue===void 0)return reduceHexValue$1("#"+numberToHex(value.red)+numberToHex(value.green)+numberToHex(value.blue));throw new PolishedError(6)}function rgba(firstValue,secondValue,thirdValue,fourthValue){if(typeof firstValue=="string"&&typeof secondValue=="number"){var rgbValue=parseToRgb(firstValue);return "rgba("+rgbValue.red+","+rgbValue.green+","+rgbValue.blue+","+secondValue+")"}else {if(typeof firstValue=="number"&&typeof secondValue=="number"&&typeof thirdValue=="number"&&typeof fourthValue=="number")return fourthValue>=1?rgb(firstValue,secondValue,thirdValue):"rgba("+firstValue+","+secondValue+","+thirdValue+","+fourthValue+")";if(typeof firstValue=="object"&&secondValue===void 0&&thirdValue===void 0&&fourthValue===void 0)return firstValue.alpha>=1?rgb(firstValue.red,firstValue.green,firstValue.blue):"rgba("+firstValue.red+","+firstValue.green+","+firstValue.blue+","+firstValue.alpha+")"}throw new PolishedError(7)}var isRgb=function(color2){return typeof color2.red=="number"&&typeof color2.green=="number"&&typeof color2.blue=="number"&&(typeof color2.alpha!="number"||typeof color2.alpha>"u")},isRgba=function(color2){return typeof color2.red=="number"&&typeof color2.green=="number"&&typeof color2.blue=="number"&&typeof color2.alpha=="number"},isHsl=function(color2){return typeof color2.hue=="number"&&typeof color2.saturation=="number"&&typeof color2.lightness=="number"&&(typeof color2.alpha!="number"||typeof color2.alpha>"u")},isHsla=function(color2){return typeof color2.hue=="number"&&typeof color2.saturation=="number"&&typeof color2.lightness=="number"&&typeof color2.alpha=="number"};function toColorString(color2){if(typeof color2!="object")throw new PolishedError(8);if(isRgba(color2))return rgba(color2);if(isRgb(color2))return rgb(color2);if(isHsla(color2))return hsla(color2);if(isHsl(color2))return hsl(color2);throw new PolishedError(8)}function curried(f,length,acc){return function(){var combined=acc.concat(Array.prototype.slice.call(arguments));return combined.length>=length?f.apply(this,combined):curried(f,length,combined)}}function curry(f){return curried(f,f.length,[])}function guard(lowerBoundary,upperBoundary,value){return Math.max(lowerBoundary,Math.min(upperBoundary,value))}function darken(amount,color2){if(color2==="transparent")return color2;var hslColor=parseToHsl(color2);return toColorString(_extends({},hslColor,{lightness:guard(0,1,hslColor.lightness-parseFloat(amount))}))}var curriedDarken=curry(darken),curriedDarken$1=curriedDarken;function lighten(amount,color2){if(color2==="transparent")return color2;var hslColor=parseToHsl(color2);return toColorString(_extends({},hslColor,{lightness:guard(0,1,hslColor.lightness+parseFloat(amount))}))}var curriedLighten=curry(lighten),curriedLighten$1=curriedLighten;function opacify(amount,color2){if(color2==="transparent")return color2;var parsedColor=parseToRgb(color2),alpha=typeof parsedColor.alpha=="number"?parsedColor.alpha:1,colorWithAlpha=_extends({},parsedColor,{alpha:guard(0,1,(alpha*100+parseFloat(amount)*100)/100)});return rgba(colorWithAlpha)}var curriedOpacify=curry(opacify),curriedOpacify$1=curriedOpacify;function transparentize(amount,color2){if(color2==="transparent")return color2;var parsedColor=parseToRgb(color2),alpha=typeof parsedColor.alpha=="number"?parsedColor.alpha:1,colorWithAlpha=_extends({},parsedColor,{alpha:guard(0,1,+(alpha*100-parseFloat(amount)*100).toFixed(2)/100)});return rgba(colorWithAlpha)}var curriedTransparentize=curry(transparentize),curriedTransparentize$1=curriedTransparentize;var color={primary:"#FF4785",secondary:"#029CFD",tertiary:"#FAFBFC",ancillary:"#22a699",orange:"#FC521F",gold:"#FFAE00",green:"#66BF3C",seafoam:"#37D5D3",purple:"#6F2CAC",ultraviolet:"#2A0481",lightest:"#FFFFFF",lighter:"#F7FAFC",light:"#EEF3F6",mediumlight:"#ECF4F9",medium:"#D9E8F2",mediumdark:"#73828C",dark:"#5C6870",darker:"#454E54",darkest:"#2E3438",border:"hsla(203, 50%, 30%, 0.15)",positive:"#66BF3C",negative:"#FF4400",warning:"#E69D00",critical:"#FFFFFF",defaultText:"#2E3438",inverseText:"#FFFFFF",positiveText:"#448028",negativeText:"#D43900",warningText:"#A15C20"},background={app:"#F6F9FC",bar:color.lightest,content:color.lightest,preview:color.lightest,gridCellSize:10,hoverable:curriedTransparentize$1(.9,color.secondary),positive:"#E1FFD4",negative:"#FEDED2",warning:"#FFF5CF",critical:"#FF4400"},typography={fonts:{base:['"Nunito Sans"',"-apple-system",'".SFNSText-Regular"','"San Francisco"',"BlinkMacSystemFont",'"Segoe UI"','"Helvetica Neue"',"Helvetica","Arial","sans-serif"].join(", "),mono:["ui-monospace","Menlo","Monaco",'"Roboto Mono"','"Oxygen Mono"','"Ubuntu Monospace"','"Source Code Pro"','"Droid Sans Mono"','"Courier New"',"monospace"].join(", ")},weight:{regular:400,bold:700},size:{s1:12,s2:14,s3:16,m1:20,m2:24,m3:28,l1:32,l2:40,l3:48,code:90}};var theme={base:"light",colorPrimary:"#FF4785",colorSecondary:"#029CFD",appBg:background.app,appContentBg:color.lightest,appPreviewBg:color.lightest,appBorderColor:color.border,appBorderRadius:4,fontBase:typography.fonts.base,fontCode:typography.fonts.mono,textColor:color.darkest,textInverseColor:color.lightest,textMutedColor:color.dark,barTextColor:color.mediumdark,barHoverColor:color.secondary,barSelectedColor:color.secondary,barBg:color.lightest,buttonBg:background.app,buttonBorder:color.medium,booleanBg:color.mediumlight,booleanSelectedBg:color.lightest,inputBg:color.lightest,inputBorder:color.border,inputTextColor:color.darkest,inputBorderRadius:4},light_default=theme;var theme2={base:"dark",colorPrimary:"#FF4785",colorSecondary:"#029CFD",appBg:"#222425",appContentBg:"#1B1C1D",appPreviewBg:color.lightest,appBorderColor:"rgba(255,255,255,.1)",appBorderRadius:4,fontBase:typography.fonts.base,fontCode:typography.fonts.mono,textColor:"#C9CDCF",textInverseColor:"#222425",textMutedColor:"#798186",barTextColor:"#798186",barHoverColor:color.secondary,barSelectedColor:color.secondary,barBg:"#292C2E",buttonBg:"#222425",buttonBorder:"rgba(255,255,255,.1)",booleanBg:"#222425",booleanSelectedBg:"#2E3438",inputBg:"#1B1C1D",inputBorder:"rgba(255,255,255,.1)",inputTextColor:color.lightest,inputBorderRadius:4},dark_default=theme2;var {window:globalWindow}=global,mkColor=color2=>({color:color2}),isColorString=color2=>typeof color2!="string"?(logger.warn(`Color passed to theme object should be a string. Instead ${color2}(${typeof color2}) was passed.`),!1):!0,isValidColorForPolished=color2=>!/(gradient|var|calc)/.test(color2),applyPolished=(type,color2)=>type==="darken"?rgba(`${curriedDarken$1(1,color2)}`,.95):type==="lighten"?rgba(`${curriedLighten$1(1,color2)}`,.95):color2,colorFactory=type=>color2=>{if(!isColorString(color2)||!isValidColorForPolished(color2))return color2;try{return applyPolished(type,color2)}catch{return color2}},lightenColor=colorFactory("lighten"),darkenColor=colorFactory("darken"),getPreferredColorScheme=()=>!globalWindow||!globalWindow.matchMedia?"light":globalWindow.matchMedia("(prefers-color-scheme: dark)").matches?"dark":"light";var themes={light:light_default,dark:dark_default,normal:light_default},preferredColorScheme=getPreferredColorScheme(),create=(vars={base:preferredColorScheme},rest)=>{let inherit={...themes[preferredColorScheme],...themes[vars.base]||{},...vars,base:themes[vars.base]?vars.base:preferredColorScheme};return {...rest,...inherit,barSelectedColor:vars.barSelectedColor||inherit.colorSecondary}};

export { __commonJS, __toESM, background, color, create, curriedOpacify$1, darkenColor, getPreferredColorScheme, light_default, lightenColor, mkColor, themes, typography };
