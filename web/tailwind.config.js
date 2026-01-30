/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        primary: {
          50: '#eff6ff',
          100: '#dbeafe',
          200: '#bfdbfe',
          300: '#93c5fd',
          400: '#60a5fa',
          500: '#3b82f6',
          600: '#2563eb',
          700: '#1d4ed8',
          800: '#1e40af',
          900: '#1e3a8a',
        },
        earthquake: {
          1: '#b3e5fc',
          2: '#81d4fa',
          3: '#4fc3f7',
          4: '#ffeb3b',
          '5weak': '#ff9800',
          '5strong': '#ff5722',
          '6weak': '#f44336',
          '6strong': '#d32f2f',
          7: '#9c27b0',
        },
      },
    },
  },
  plugins: [],
}
