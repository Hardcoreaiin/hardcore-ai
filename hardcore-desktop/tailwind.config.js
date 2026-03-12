/** @type {import('tailwindcss').Config} */
export default {
    content: [
        "./index.html",
        "./src/**/*.{js,ts,jsx,tsx}",
    ],
    theme: {
        extend: {
            colors: {
                'bg-dark': '#000000',
                'bg-surface': '#0a0a0a',
                'bg-elevated': '#111111',
                'bg-hover': '#1a1a1a',
                'border': '#1f1f1f',
                'border-subtle': '#151515',
                'text-primary': '#e0e0e0',
                'text-secondary': '#888888',
                'text-muted': '#555555',
                'accent-primary': '#7c3aed',
                'accent-secondary': '#5b21b6',
                'accent-hover': '#8b5cf6',
                'success': '#10b981',
                'warning': '#f59e0b',
                'error': '#dc2626',
            },
            fontFamily: {
                sans: ['JetBrains Mono', 'Fira Code', 'Consolas', 'monospace'],
                mono: ['JetBrains Mono', 'Fira Code', 'Consolas', 'monospace'],
            },
            fontSize: {
                'xs': '11px',
                'sm': '12px',
                'base': '13px',
                'lg': '14px',
            },
        },
    },
    plugins: [],
}
