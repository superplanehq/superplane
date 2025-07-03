export interface StoredData {
  [key: string]: any;
}
export type Replacer = (key: any, value: any) => string | String[] | number[];
export type Reviver = (key: string, value: any) => any;
export type EachFn = (key: any, data: any) => false | any;
export type TransactFn = (data: any) => any | undefined;

type BaseSet = (key: any, data: any) => any;
type BaseGet = (key: any) => any;
type BaseSetAll = (obj: Object) => StoredData;
type BaseGetAll = () => StoredData;
type BaseTransact = (fn: EachFn, value?: any) => StoredData;
type BaseClear = (clear: false) => StoreBase;
export type Base = BaseSet & BaseGet & BaseSetAll & BaseGetAll & BaseTransact & BaseClear;

export interface StoreAPI {
  clear(): StoreBase;
  clearAll(): StoreBase;
  each(callback: EachFn): StoreBase;
  get(key: any, alt?: any|Reviver): any;
  getAll(fillObj?: StoredData): StoredData;
  has(key: any): boolean;
  isFake(force?: boolean): boolean;
  keys(fillList?: string[]): string[];
  namespace(namespace: string, singleArea?: true, delim?: string): StoreType;
  remove(key: any, alt?: any|Reviver): any;
  set(key: any, data: any, overwrite?: boolean|Replacer): any;
  setAll(data: Object, overwrite?: boolean|Replacer): StoredData;
  add(key: any, data: any): any;
  size(): number;
  transact(key: any, fn: TransactFn, alt?: any|Reviver): StoreBase;
  area(id: string, area: Storage): StoreBase
}

export type StoreBase = StoreAPI & Base;

// these are not guaranteed to be stable across minor versions
// but historically, they have been pretty much so
export interface DeveloperTools {
  readonly version: string;
  readonly areas: { [name: string]: Storage };
  readonly apis: { [name: string]: StoreAPI };
  nsdelim: string;
  revive: Reviver;
  replace: Replacer;
  readonly fn: (name: string, fn: Function) => void;
  storeAPI: StoreAPI;
  get: (area: Storage, key: string) => string;
  set: (area: Storage, key: string, string: string) => void;
  remove: (area: Storage, key: string) => void;
  key: (area: Storage, i: number) => string;
  length: (area: Storage) => number;
  clear: (area: Storage) => void;
  parse: (s: string, fn?: Reviver) => any;
  stringify: (d: any, fn?: Replacer) => string;
  inherit: (api: StoreAPI, o: object) => object;
}

export type StoreType = StoreBase & {
  local: StoreBase;
  session: StoreBase;
  page: StoreBase;
  readonly _: DeveloperTools,
};

declare const store: StoreType
export default store
