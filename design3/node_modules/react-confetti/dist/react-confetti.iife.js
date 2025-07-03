var ReactConfetti = (function (jsxRuntime, React, tweens) {
    'use strict';

    function _interopNamespaceDefault(e) {
        var n = Object.create(null);
        if (e) {
            Object.keys(e).forEach(function (k) {
                if (k !== 'default') {
                    var d = Object.getOwnPropertyDescriptor(e, k);
                    Object.defineProperty(n, k, d.get ? d : {
                        enumerable: true,
                        get: function () { return e[k]; }
                    });
                }
            });
        }
        n.default = e;
        return Object.freeze(n);
    }

    var tweens__namespace = /*#__PURE__*/_interopNamespaceDefault(tweens);

    function degreesToRads(degrees) {
        return (degrees * Math.PI) / 180;
    }
    function randomRange(min, max) {
        return min + Math.random() * (max - min);
    }
    function randomInt(min, max) {
        return Math.floor(min + Math.random() * (max - min + 1));
    }

    var ParticleShape;
    (function (ParticleShape) {
        ParticleShape[ParticleShape["Circle"] = 0] = "Circle";
        ParticleShape[ParticleShape["Square"] = 1] = "Square";
        ParticleShape[ParticleShape["Strip"] = 2] = "Strip";
    })(ParticleShape || (ParticleShape = {}));
    var RotationDirection;
    (function (RotationDirection) {
        RotationDirection[RotationDirection["Positive"] = 1] = "Positive";
        RotationDirection[RotationDirection["Negative"] = -1] = "Negative";
    })(RotationDirection || (RotationDirection = {}));
    const DEFAULT_FRAME_TIME = 1000 / 60;
    class Particle {
        constructor(context, getOptions, x, y) {
            this.getOptions = getOptions;
            const { colors, initialVelocityX, initialVelocityY } = this.getOptions();
            this.context = context;
            this.x = x;
            this.y = y;
            this.w = randomRange(5, 20);
            this.h = randomRange(5, 20);
            this.radius = randomRange(5, 10);
            this.vx =
                typeof initialVelocityX === 'number'
                    ? randomRange(-initialVelocityX, initialVelocityX)
                    : randomRange(initialVelocityX.min, initialVelocityX.max);
            this.vy =
                typeof initialVelocityY === 'number'
                    ? randomRange(-initialVelocityY, 0)
                    : randomRange(initialVelocityY.min, initialVelocityY.max);
            this.shape = randomInt(0, 2);
            this.angle = degreesToRads(randomRange(0, 360));
            this.angularSpin = randomRange(-0.2, 0.2);
            this.color = colors[Math.floor(Math.random() * colors.length)];
            this.rotateY = randomRange(0, 1);
            this.rotationDirection = randomRange(0, 1)
                ? RotationDirection.Positive
                : RotationDirection.Negative;
        }
        update(elapsed) {
            const { gravity, wind, friction, opacity, drawShape } = this.getOptions();
            const frameTimeMultiplier = elapsed / DEFAULT_FRAME_TIME;
            this.x += this.vx * frameTimeMultiplier;
            this.y += this.vy * frameTimeMultiplier;
            this.vy += gravity * frameTimeMultiplier;
            this.vx += wind * frameTimeMultiplier;
            this.vx *= friction ** frameTimeMultiplier;
            this.vy *= friction ** frameTimeMultiplier;
            if (this.rotateY >= 1 &&
                this.rotationDirection === RotationDirection.Positive) {
                this.rotationDirection = RotationDirection.Negative;
            }
            else if (this.rotateY <= -1 &&
                this.rotationDirection === RotationDirection.Negative) {
                this.rotationDirection = RotationDirection.Positive;
            }
            const rotateDelta = 0.1 * this.rotationDirection * frameTimeMultiplier;
            this.rotateY += rotateDelta;
            this.angle += this.angularSpin;
            this.context.save();
            this.context.translate(this.x, this.y);
            this.context.rotate(this.angle);
            this.context.scale(1, this.rotateY);
            this.context.rotate(this.angle);
            this.context.beginPath();
            this.context.fillStyle = this.color;
            this.context.strokeStyle = this.color;
            this.context.globalAlpha = opacity;
            this.context.lineCap = 'round';
            this.context.lineWidth = 2;
            if (drawShape && typeof drawShape === 'function') {
                drawShape.call(this, this.context);
            }
            else {
                switch (this.shape) {
                    case ParticleShape.Circle: {
                        this.context.beginPath();
                        this.context.arc(0, 0, this.radius, 0, 2 * Math.PI);
                        this.context.fill();
                        break;
                    }
                    case ParticleShape.Square: {
                        this.context.fillRect(-this.w / 2, -this.h / 2, this.w, this.h);
                        break;
                    }
                    case ParticleShape.Strip: {
                        this.context.fillRect(-this.w / 6, -this.h / 2, this.w / 3, this.h);
                        break;
                    }
                }
            }
            this.context.closePath();
            this.context.restore();
        }
    }

    class ParticleGenerator {
        constructor(canvas, getOptions) {
            this.x = 0;
            this.y = 0;
            this.w = 0;
            this.h = 0;
            this.lastNumberOfPieces = 0;
            this.tweenProgress = 0;
            this.tweenFrom = 0;
            this.particles = [];
            this.particlesGenerated = 0;
            this.removeParticleAt = (i) => {
                this.particles.splice(i, 1);
            };
            this.getParticle = () => {
                const newParticleX = randomRange(this.x, this.w + this.x);
                const newParticleY = randomRange(this.y, this.h + this.y);
                return new Particle(this.context, this.getOptions, newParticleX, newParticleY);
            };
            this.animate = (elapsed) => {
                const { canvas, context, particlesGenerated, lastNumberOfPieces } = this;
                const { run, recycle, numberOfPieces, debug, tweenFunction, tweenDuration, } = this.getOptions();
                if (!run) {
                    return false;
                }
                const nP = this.particles.length;
                const activeCount = recycle ? nP : particlesGenerated;
                // Initial population
                if (activeCount < numberOfPieces) {
                    // Use the numberOfPieces prop as a key to reset the easing timing
                    if (lastNumberOfPieces !== numberOfPieces) {
                        this.tweenProgress = 0;
                        this.tweenFrom = activeCount;
                        this.lastNumberOfPieces = numberOfPieces;
                    }
                    // Clamp tweenProgress between 0 and tweenDuration
                    this.tweenProgress = Math.min(tweenDuration, Math.max(0, this.tweenProgress + elapsed));
                    const tweenedVal = tweenFunction(this.tweenProgress, this.tweenFrom, numberOfPieces, tweenDuration);
                    const numToAdd = Math.round(tweenedVal - activeCount);
                    for (let i = 0; i < numToAdd; i++) {
                        this.particles.push(this.getParticle());
                    }
                    this.particlesGenerated += numToAdd;
                }
                if (debug) {
                    // Draw debug text
                    context.font = '12px sans-serif';
                    context.fillStyle = '#333';
                    context.textAlign = 'right';
                    context.fillText(`Particles: ${nP}`, canvas.width - 10, canvas.height - 20);
                }
                // Maintain the population, iterating backwards to prevent issues when removing particles
                for (let i = this.particles.length - 1; i >= 0; i--) {
                    const p = this.particles[i];
                    // Update each particle's position
                    p.update(elapsed);
                    // Prune the off-canvas particles
                    if (p.y > canvas.height ||
                        p.y < -100 ||
                        p.x > canvas.width + 100 ||
                        p.x < -100) {
                        if (recycle && activeCount <= numberOfPieces) {
                            // Replace the particle with a brand new one
                            this.particles[i] = this.getParticle();
                        }
                        else {
                            this.removeParticleAt(i);
                        }
                    }
                }
                return nP > 0 || activeCount < numberOfPieces;
            };
            this.canvas = canvas;
            const ctx = this.canvas.getContext('2d');
            if (!ctx) {
                throw new Error('Could not get canvas context');
            }
            this.context = ctx;
            this.getOptions = getOptions;
        }
    }

    const confettiDefaults = {
        width: typeof window !== 'undefined' ? window.innerWidth : 300,
        height: typeof window !== 'undefined' ? window.innerHeight : 200,
        numberOfPieces: 200,
        friction: 0.99,
        wind: 0,
        gravity: 0.1,
        initialVelocityX: 4,
        initialVelocityY: 10,
        colors: [
            '#f44336',
            '#e91e63',
            '#9c27b0',
            '#673ab7',
            '#3f51b5',
            '#2196f3',
            '#03a9f4',
            '#00bcd4',
            '#009688',
            '#4CAF50',
            '#8BC34A',
            '#CDDC39',
            '#FFEB3B',
            '#FFC107',
            '#FF9800',
            '#FF5722',
            '#795548',
        ],
        opacity: 1.0,
        debug: false,
        tweenFunction: tweens__namespace.easeInOutQuad,
        tweenDuration: 5000,
        recycle: true,
        run: true,
    };
    class Confetti {
        constructor(canvas, opts) {
            this.lastFrameTime = 0;
            this.setOptionsWithDefaults = (opts) => {
                const computedConfettiDefaults = {
                    confettiSource: {
                        x: 0,
                        y: 0,
                        w: this.canvas.width,
                        h: 0,
                    },
                };
                this._options = {
                    ...computedConfettiDefaults,
                    ...confettiDefaults,
                    ...opts,
                };
                Object.assign(this, opts.confettiSource);
            };
            this.update = (timestamp = 0) => {
                const { options: { run, onConfettiComplete, frameRate }, canvas, context, } = this;
                // Cap elapsed time to 50ms to prevent large time steps
                const elapsed = Math.min(timestamp - this.lastFrameTime, 50);
                // Throttle the frame rate if set
                if (frameRate && elapsed < 1000 / frameRate) {
                    this.rafId = requestAnimationFrame(this.update);
                    return;
                }
                this.lastFrameTime = timestamp - (frameRate ? elapsed % frameRate : 0);
                if (run) {
                    context.fillStyle = 'white';
                    context.clearRect(0, 0, canvas.width, canvas.height);
                }
                if (this.generator.animate(elapsed)) {
                    this.rafId = requestAnimationFrame(this.update);
                }
                else {
                    if (onConfettiComplete &&
                        typeof onConfettiComplete === 'function' &&
                        this.generator.particlesGenerated > 0) {
                        onConfettiComplete.call(this, this);
                    }
                    this._options.run = false;
                }
            };
            this.reset = () => {
                if (this.generator && this.generator.particlesGenerated > 0) {
                    this.generator.particlesGenerated = 0;
                    this.generator.particles = [];
                    this.generator.lastNumberOfPieces = 0;
                }
            };
            this.stop = () => {
                this.options = { run: false };
                if (this.rafId) {
                    cancelAnimationFrame(this.rafId);
                    this.rafId = undefined;
                }
            };
            this.canvas = canvas;
            const ctx = this.canvas.getContext('2d');
            if (!ctx) {
                throw new Error('Could not get canvas context');
            }
            this.context = ctx;
            this.generator = new ParticleGenerator(this.canvas, () => this.options);
            this.options = opts;
            this.update();
        }
        get options() {
            return this._options;
        }
        set options(opts) {
            const lastRunState = this._options?.run;
            const lastRecycleState = this._options?.recycle;
            this.setOptionsWithDefaults(opts);
            if (this.generator) {
                Object.assign(this.generator, this.options.confettiSource);
                if (typeof opts.recycle === 'boolean' &&
                    opts.recycle &&
                    lastRecycleState === false) {
                    this.generator.lastNumberOfPieces = this.generator.particles.length;
                }
            }
            if (typeof opts.run === 'boolean' && opts.run && lastRunState === false) {
                this.update();
            }
        }
    }

    const ref = React.createRef();
    class ReactConfettiInternal extends React.Component {
        constructor(props) {
            super(props);
            this.canvas = React.createRef();
            this.canvas = props.canvasRef || ref;
        }
        componentDidMount() {
            if (this.canvas.current) {
                const opts = extractCanvasProps(this.props)[0];
                this.confetti = new Confetti(this.canvas.current, opts);
            }
        }
        componentDidUpdate() {
            const confettiOptions = extractCanvasProps(this.props)[0];
            if (this.confetti) {
                this.confetti.options = confettiOptions;
            }
        }
        componentWillUnmount() {
            if (this.confetti) {
                this.confetti.stop();
            }
            this.confetti = undefined;
        }
        render() {
            const [confettiOptions, passedProps] = extractCanvasProps(this.props);
            const canvasStyles = {
                zIndex: 2,
                position: 'absolute',
                pointerEvents: 'none',
                top: 0,
                left: 0,
                bottom: 0,
                right: 0,
                ...passedProps.style,
            };
            return (jsxRuntime.jsx("canvas", { width: confettiOptions.width, height: confettiOptions.height, ref: this.canvas, ...passedProps, style: canvasStyles }));
        }
    }
    ReactConfettiInternal.defaultProps = {
        ...confettiDefaults,
    };
    ReactConfettiInternal.displayName = 'ReactConfetti';
    function extractCanvasProps(props) {
        const confettiOptions = {};
        const refs = {};
        const rest = {};
        const confettiOptionKeys = [
            ...Object.keys(confettiDefaults),
            'confettiSource',
            'drawShape',
            'onConfettiComplete',
            'frameRate',
        ];
        const refProps = ['canvasRef'];
        for (const prop in props) {
            const val = props[prop];
            if (confettiOptionKeys.includes(prop)) {
                confettiOptions[prop] = val;
            }
            else if (refProps.includes(prop)) {
                refProps[prop] = val;
            }
            else {
                rest[prop] = val;
            }
        }
        return [confettiOptions, rest, refs];
    }
    const ReactConfetti = React.forwardRef((props, ref) => jsxRuntime.jsx(ReactConfettiInternal, { canvasRef: ref, ...props }));

    return ReactConfetti;

})(jsxRuntime, React, tweens);
//# sourceMappingURL=react-confetti.iife.js.map
