export type ManifestJSONValue =
  | null
  | boolean
  | number
  | string
  | ManifestJSONValue[]
  | { [key: string]: ManifestJSONValue };

export interface ManifestV1 {
  apiVersion: "spx/v1";
  kind: "extension";
  metadata: ExtensionMetadata;
  runtime: RuntimeDescriptor;
  integrations: IntegrationBlock[];
  components: ComponentBlock[];
  triggers: TriggerBlock[];
}

export interface ExtensionMetadata {
  id: string;
  name: string;
  version: string;
  description?: string;
}

export interface RuntimeDescriptor {
  profile: "portable-web-v1" | string;
}

export interface IntegrationBlock {
  name: string;
  label: string;
  icon: string;
  description: string;
  instructions?: string;
  configuration: ConfigurationField[];
  actions: ActionDefinition[];
  resourceTypes: string[];
}

export interface ComponentBlock {
  name: string;
  integration?: string;
  label: string;
  description: string;
  icon: string;
  color: string;
  outputChannels: OutputChannel[];
  configuration: ConfigurationField[];
  actions: ActionDefinition[];
}

export interface TriggerBlock {
  name: string;
  integration?: string;
  label: string;
  description: string;
  icon: string;
  color: string;
  configuration: ConfigurationField[];
  actions: ActionDefinition[];
}

export interface OutputChannel {
  name: string;
  label: string;
  description?: string;
}

export interface ActionDefinition {
  name: string;
  description: string;
  userAccessible?: boolean;
  parameters: ConfigurationField[];
}

export type ConfigurationField =
  | StringField
  | TextField
  | ExpressionField
  | XMLField
  | NumberField
  | BooleanField
  | SelectField
  | MultiSelectField
  | ListField
  | ObjectField
  | TimeField
  | DateField
  | DateTimeField
  | TimezoneField
  | DaysOfWeekField
  | TimeRangeField
  | DayInYearField
  | CronField
  | UserField
  | RoleField
  | GroupField
  | IntegrationResourceField
  | AnyPredicateListField
  | GitRefField
  | SecretKeyField;

export type ConfigurationFieldType =
  | "string"
  | "text"
  | "expression"
  | "xml"
  | "number"
  | "boolean"
  | "select"
  | "multi-select"
  | "list"
  | "object"
  | "time"
  | "date"
  | "datetime"
  | "timezone"
  | "days-of-week"
  | "time-range"
  | "day-in-year"
  | "cron"
  | "user"
  | "role"
  | "group"
  | "integration-resource"
  | "any-predicate-list"
  | "git-ref"
  | "secret-key";

export interface BaseConfigurationField<
  TType extends ConfigurationFieldType,
  TDefault = ManifestJSONValue,
> {
  name: string;
  label: string;
  placeholder?: string;
  type: TType;
  description?: string;
  required?: boolean;
  default?: TDefault;
  togglable?: boolean;
  disallowExpression?: boolean;
  sensitive?: boolean;
  typeOptions?: TypeOptions;
  visibilityConditions?: VisibilityCondition[];
  requiredConditions?: RequiredCondition[];
  validationRules?: ValidationRule[];
}

export interface StringField extends BaseConfigurationField<"string", string> {
  typeOptions?: {
    string?: StringTypeOptions;
  };
}

export interface TextField extends BaseConfigurationField<"text", string> {
  typeOptions?: {
    text?: TextTypeOptions;
  };
}

export interface ExpressionField
  extends BaseConfigurationField<"expression", string> {
  typeOptions?: {
    expression?: ExpressionTypeOptions;
  };
}

export interface XMLField extends BaseConfigurationField<"xml", string> {}

export interface NumberField extends BaseConfigurationField<"number", number> {
  typeOptions?: {
    number?: NumberTypeOptions;
  };
}

export interface BooleanField
  extends BaseConfigurationField<"boolean", boolean> {}

export interface SelectField extends BaseConfigurationField<"select", string> {
  typeOptions: {
    select: SelectTypeOptions;
  };
}

export interface MultiSelectField
  extends BaseConfigurationField<"multi-select", string[]> {
  typeOptions: {
    multiSelect: MultiSelectTypeOptions;
  };
}

export interface ListField
  extends BaseConfigurationField<"list", ManifestJSONValue[]> {
  typeOptions: {
    list: ListTypeOptions;
  };
}

export interface ObjectField
  extends BaseConfigurationField<"object", ManifestJSONValue> {
  typeOptions: {
    object: ObjectTypeOptions;
  };
}

export interface TimeField extends BaseConfigurationField<"time", string> {
  typeOptions?: {
    time?: TimeTypeOptions;
  };
}

export interface DateField extends BaseConfigurationField<"date", string> {
  typeOptions?: {
    date?: DateTypeOptions;
  };
}

export interface DateTimeField
  extends BaseConfigurationField<"datetime", string> {
  typeOptions?: {
    dateTime?: DateTimeTypeOptions;
  };
}

export interface TimezoneField
  extends BaseConfigurationField<"timezone", string> {
  typeOptions?: {
    timezone?: TimezoneTypeOptions;
  };
}

export interface DaysOfWeekField
  extends BaseConfigurationField<"days-of-week", string[]> {}

export interface TimeRangeField
  extends BaseConfigurationField<
    "time-range",
    { start: string; end: string }
  > {}

export interface DayInYearField
  extends BaseConfigurationField<"day-in-year", string> {
  typeOptions?: {
    dayInYear?: DayInYearTypeOptions;
  };
}

export interface CronField extends BaseConfigurationField<"cron", string> {
  typeOptions?: {
    cron?: CronTypeOptions;
  };
}

export interface UserField extends BaseConfigurationField<"user", string> {}

export interface RoleField extends BaseConfigurationField<"role", string> {}

export interface GroupField extends BaseConfigurationField<"group", string> {}

export interface IntegrationResourceField
  extends BaseConfigurationField<"integration-resource", string | string[]> {
  typeOptions: {
    resource: ResourceTypeOptions;
  };
}

export interface AnyPredicateListField
  extends BaseConfigurationField<"any-predicate-list", Predicate[]> {
  typeOptions: {
    anyPredicateList: AnyPredicateListTypeOptions;
  };
}

export interface GitRefField
  extends BaseConfigurationField<"git-ref", string> {}

export interface SecretKeyField
  extends BaseConfigurationField<"secret-key", string> {}

export interface TypeOptions {
  number?: NumberTypeOptions;
  string?: StringTypeOptions;
  text?: TextTypeOptions;
  expression?: ExpressionTypeOptions;
  select?: SelectTypeOptions;
  multiSelect?: MultiSelectTypeOptions;
  resource?: ResourceTypeOptions;
  list?: ListTypeOptions;
  anyPredicateList?: AnyPredicateListTypeOptions;
  object?: ObjectTypeOptions;
  time?: TimeTypeOptions;
  date?: DateTypeOptions;
  dateTime?: DateTimeTypeOptions;
  dayInYear?: DayInYearTypeOptions;
  cron?: CronTypeOptions;
  timezone?: TimezoneTypeOptions;
}

export interface NumberTypeOptions {
  min?: number;
  max?: number;
}

export interface StringTypeOptions {
  minLength?: number;
  maxLength?: number;
}

export interface ExpressionTypeOptions {
  minLength?: number;
  maxLength?: number;
}

export interface TextTypeOptions {
  minLength?: number;
  maxLength?: number;
}

export interface TimeTypeOptions {
  format?: string;
}

export interface DateTypeOptions {
  format?: string;
}

export interface DateTimeTypeOptions {
  format?: string;
}

export interface DayInYearTypeOptions {
  format?: string;
}

export interface CronTypeOptions {
  allowedFields?: string[];
}

export interface TimezoneTypeOptions {}

export interface SelectTypeOptions {
  options: FieldOption[];
}

export interface MultiSelectTypeOptions {
  options: FieldOption[];
}

export interface ListTypeOptions {
  itemDefinition?: ListItemDefinition;
  itemLabel?: string;
  maxItems?: number;
}

export interface ObjectTypeOptions {
  schema: ConfigurationField[];
}

export interface ResourceTypeOptions {
  type: string;
  useNameAsValue?: boolean;
  multi?: boolean;
  parameters?: ParameterRef[];
}

export interface ParameterRef {
  name: string;
  value?: string;
  valueFrom?: ParameterValueFrom;
}

export interface ParameterValueFrom {
  field: string;
}

export interface FieldOption {
  label: string;
  value: string;
}

export interface ListItemDefinition {
  type: string;
  schema?: ConfigurationField[];
}

export interface VisibilityCondition {
  field: string;
  values: string[];
}

export interface RequiredCondition {
  field: string;
  values: string[];
}

export type ValidationRuleType =
  | "less_than"
  | "greater_than"
  | "equal"
  | "not_equal"
  | "max_length"
  | "min_length";

export interface ValidationRule {
  type: ValidationRuleType;
  compareWith?: string;
  value?: ManifestJSONValue;
  message: string;
}

export type PredicateType = "equals" | "notEquals" | "matches";

export interface Predicate {
  type: PredicateType;
  value: string;
}

export interface AnyPredicateListTypeOptions {
  operators: FieldOption[];
}
