var StorybookError=class extends Error{constructor(){super(...arguments);this.data={};this.documentation=!1;this.fromStorybook=!0;}get fullErrorCode(){let paddedCode=String(this.code).padStart(4,"0");return `SB_${this.category}_${paddedCode}`}get name(){let errorName=this.constructor.name;return `${this.fullErrorCode} (${errorName})`}get message(){let page;return this.documentation===!0?page=`https://storybook.js.org/error/${this.fullErrorCode}`:typeof this.documentation=="string"?page=this.documentation:Array.isArray(this.documentation)&&(page=`
${this.documentation.map(doc=>`	- ${doc}`).join(`
`)}`),`${this.template()}${page!=null?`

More info: ${page}
`:""}`}};

export { StorybookError };
