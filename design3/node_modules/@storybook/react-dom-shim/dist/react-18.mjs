import React, { useRef, useLayoutEffect } from 'react';
import ReactDOM from 'react-dom/client';

var nodes=new Map,WithCallback=({callback,children})=>{let once=useRef();return useLayoutEffect(()=>{once.current!==callback&&(once.current=callback,callback());},[callback]),children},renderElement=async(node,el)=>{let root=await getReactRoot(el);return new Promise(resolve=>{root.render(React.createElement(WithCallback,{callback:()=>resolve(null)},node));})},unmountElement=(el,shouldUseNewRootApi)=>{let root=nodes.get(el);root&&(root.unmount(),nodes.delete(el));},getReactRoot=async el=>{let root=nodes.get(el);return root||(root=ReactDOM.createRoot(el),nodes.set(el,root)),root};

export { renderElement, unmountElement };
