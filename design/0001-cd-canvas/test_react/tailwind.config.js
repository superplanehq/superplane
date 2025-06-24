/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ["./src/**/*.{html,js,ts,jsx,tsx}"],
  plugins: [],
  theme: {
    extend: {
      fontFamily: {
        // Use Fakt Pro for all text
        sans: ['"Fakt Pro"', '"sans-serif"'],
        // Keep the custom fakt family for specific use cases
        fakt: ['"Fakt Pro"', '"sans-serif"'],
        // Add fallback serif font
        serif: ['Georgia', 'Cambria', '"Times New Roman"', 'Times', 'serif'],
        // Add monospace font for code
        mono: ['"Jetbrains Mono"', 'monospace'],
      },
    },
  },
  corePlugins: {
    preflight: true,
  },
};