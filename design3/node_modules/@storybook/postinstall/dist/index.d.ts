interface PostinstallContext {
    root: any;
    api: any;
}
declare function addPreset(preset: string, presetOptions: any, { api, root }: PostinstallContext): void;

type Deps = Record<string, string>;
interface PackageJson {
    dependencies?: Deps;
    devDependencies?: Deps;
}
declare const getFrameworks: ({ dependencies, devDependencies }: PackageJson) => string[];

export { getFrameworks, addPreset as presetsAddPreset };
