import { global } from '@storybook/global';
import React, { StrictMode, Fragment, Component } from 'react';
import { unmountElement, renderElement } from '@storybook/react-dom-shim';

var {FRAMEWORK_OPTIONS}=global,render=(args,context)=>{let{id,component:Component}=context;if(!Component)throw new Error(`Unable to render story ${id} as the component annotation is missing from the default export`);return React.createElement(Component,{...args})},ErrorBoundary=class extends Component{constructor(){super(...arguments);this.state={hasError:!1};}static getDerivedStateFromError(){return {hasError:!0}}componentDidMount(){let{hasError}=this.state,{showMain}=this.props;hasError||showMain();}componentDidCatch(err){let{showException}=this.props;showException(err);}render(){let{hasError}=this.state,{children}=this.props;return hasError?null:children}},Wrapper=FRAMEWORK_OPTIONS?.strictMode?StrictMode:Fragment;async function renderToCanvas({storyContext,unboundStoryFn,showMain,showException,forceRemount},canvasElement){let content=React.createElement(ErrorBoundary,{showMain,showException},React.createElement(unboundStoryFn,{...storyContext})),element=Wrapper?React.createElement(Wrapper,null,content):content;return forceRemount&&unmountElement(canvasElement),await renderElement(element,canvasElement),()=>unmountElement(canvasElement)}

export { render, renderToCanvas };
