import React from 'react';

declare enum ParticleShape {
    Circle = 0,
    Square = 1,
    Strip = 2
}
declare enum RotationDirection {
    Positive = 1,
    Negative = -1
}
declare class Particle {
    constructor(context: CanvasRenderingContext2D, getOptions: () => IConfettiOptions, x: number, y: number);
    context: CanvasRenderingContext2D;
    radius: number;
    x: number;
    y: number;
    w: number;
    h: number;
    vx: number;
    vy: number;
    shape: ParticleShape;
    angle: number;
    angularSpin: number;
    color: string;
    rotateY: number;
    rotationDirection: RotationDirection;
    getOptions: () => IConfettiOptions;
    update(elapsed: number): void;
}

interface IRect {
    x: number;
    y: number;
    w: number;
    h: number;
}

interface IParticleGenerator extends IRect {
    removeParticleAt: (index: number) => void;
    getParticle: () => void;
    animate: (elapsed: number) => boolean;
    particles: Particle[];
    particlesGenerated: number;
}
declare class ParticleGenerator implements IParticleGenerator {
    constructor(canvas: HTMLCanvasElement, getOptions: () => IConfettiOptions);
    canvas: HTMLCanvasElement;
    context: CanvasRenderingContext2D;
    getOptions: () => IConfettiOptions;
    x: number;
    y: number;
    w: number;
    h: number;
    lastNumberOfPieces: number;
    tweenProgress: number;
    tweenFrom: number;
    particles: Particle[];
    particlesGenerated: number;
    removeParticleAt: (i: number) => void;
    getParticle: () => Particle;
    animate: (elapsed: number) => boolean;
}

interface IConfettiOptions {
    /**
     * Width of the component
     * @default window.width
     */
    width: number;
    /**
     * Height of the component
     * @default window.height
     */
    height: number;
    /**
     * Max number of confetti pieces to render.
     * @default 200
     */
    numberOfPieces: number;
    /**
     * Slows movement of pieces. (lower number = slower confetti)
     * @default 0.99
     */
    friction: number;
    /**
     * Blows confetti along the X axis.
     * @default 0
     */
    wind: number;
    /**
     * How fast it falls (pixels per frame)
     * @default 0.1
     */
    gravity: number;
    /**
     * How fast the confetti is emitted horizontally
     * @default 4
     */
    initialVelocityX: {
        min: number;
        max: number;
    } | number;
    /**
     * How fast the confetti is emitted vertically
     * @default 10
     */
    initialVelocityY: {
        min: number;
        max: number;
    } | number;
    /**
     * Array of colors to choose from.
     */
    colors: string[];
    /**
     * Opacity of the confetti.
     * @default 1
     */
    opacity: number;
    /**
     * If false, only numberOfPieces will be emitted and then stops. If true, when a confetto goes offscreen, a new one will be emitted.
     * @default true
     */
    recycle: boolean;
    /**
     * If false, stops the requestAnimationFrame loop.
     * @default true
     */
    run: boolean;
    /**
     * The frame rate of the animation. If set, the animation will be throttled to that frame rate.
     * @default undefined
     */
    frameRate?: number;
    /**
     * Renders some debug text on the canvas.
     * @default false
     */
    debug: boolean;
    /**
     * A Rect defining the area where the confetti will spawn.
     * @default {
     *   x: 0,
     *   y: 0,
     *   w: canvas.width,
     *   h: 0
     * }
     */
    confettiSource: IRect;
    /**
     * Controls the rate at which confetti is spawned.
     * @default easeInOutQuad
     */
    tweenFunction: (currentTime: number, currentValue: number, targetValue: number, duration: number, s?: number) => number;
    /**
     * Number of milliseconds it should take to spawn numberOfPieces.
     * @default 5000
     */
    tweenDuration: number;
    /**
     * Function to draw your own confetti shapes.
     */
    drawShape?: (context: CanvasRenderingContext2D) => void;
    /**
     * Function called when all confetti has fallen off-canvas.
     */
    onConfettiComplete?: (confettiInstance?: Confetti) => void;
}
declare class Confetti {
    constructor(canvas: HTMLCanvasElement, opts: Partial<IConfettiOptions>);
    canvas: HTMLCanvasElement;
    context: CanvasRenderingContext2D;
    _options: IConfettiOptions;
    generator: ParticleGenerator;
    rafId?: number;
    lastFrameTime: number;
    get options(): Partial<IConfettiOptions>;
    set options(opts: Partial<IConfettiOptions>);
    setOptionsWithDefaults: (opts: Partial<IConfettiOptions>) => void;
    update: (timestamp?: number) => void;
    reset: () => void;
    stop: () => void;
}

declare const ReactConfetti: React.ForwardRefExoticComponent<Partial<IConfettiOptions> & React.CanvasHTMLAttributes<HTMLCanvasElement> & {
    canvasRef?: React.Ref<HTMLCanvasElement>;
} & React.RefAttributes<HTMLCanvasElement>>;

export { type IConfettiOptions, ReactConfetti as default };
