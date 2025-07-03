var makeDecorator=({name,parameterName,wrapper,skipIfNoParametersOrOptions=!1})=>{let decorator=options=>(storyFn,context)=>{let parameters=context.parameters&&context.parameters[parameterName];return parameters&&parameters.disable||skipIfNoParametersOrOptions&&!options&&!parameters?storyFn(context):wrapper(storyFn,context,{options,parameters})};return (...args)=>typeof args[0]=="function"?decorator()(...args):(...innerArgs)=>{if(innerArgs.length>1)return args.length>1?decorator(args)(...innerArgs):decorator(...args)(...innerArgs);throw new Error(`Passing stories directly into ${name}() is not allowed,
        instead use addDecorator(${name}) and pass options with the '${parameterName}' parameter`)}};

export { makeDecorator };
