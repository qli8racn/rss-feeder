/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      colors: {
        surface: {
          base: '#0d1117',
          raised: '#1e2733',
        },
      },
      fontSize: {
        micro: ['10px', { lineHeight: '1.4' }],
        caption: ['11px', { lineHeight: '1.4' }],
        small: ['12px', { lineHeight: '1.4' }],
        body: ['13px', { lineHeight: '1.5' }],
      },
    },
  },
  plugins: [],
}

