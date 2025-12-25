/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        background: 'hsl(var(--color-background))',
        foreground: 'hsl(var(--color-foreground))',
        primary: 'hsl(var(--color-primary))',
        secondary: 'hsl(var(--color-secondary))',
        muted: 'hsl(var(--color-muted))',
        accent: 'hsl(var(--color-accent))',
        destructive: 'hsl(var(--color-destructive))',
        border: 'hsl(var(--color-border))',
      },
      borderRadius: {
        DEFAULT: 'var(--radius)',
      },
    },
  },
  plugins: [],
}

