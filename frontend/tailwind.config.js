/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        twitter: {
          blue: '#1DA1F2',
          darkblue: '#1A91DA',
          lightblue: '#AAB8C2',
          dark: '#14171A',
          darkgray: '#657786',
          lightgray: '#AAB8C2',
          extralightgray: '#E1E8ED',
          white: '#F7F9FA',
        }
      },
      fontFamily: {
        sans: ['Inter', 'system-ui', 'sans-serif'],
      },
    },
  },
  plugins: [],
}