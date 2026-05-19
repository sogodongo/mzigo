import type { Config } from "tailwindcss";

const config: Config = {
  content: [
    "./app/**/*.{ts,tsx}",
    "./components/**/*.{ts,tsx}",
    "./lib/**/*.{ts,tsx}",
  ],
  theme: {
    extend: {
      fontFamily: {
        mono: ["'JetBrains Mono'", "ui-monospace", "monospace"],
        sans: ["'DM Sans'", "ui-sans-serif", "system-ui"],
      },
      colors: {
        // Mzigo design system: deep slate base with amber accent.
        // Chosen to evoke infrastructure tooling: serious, precise, readable.
        // Amber reads as "alert" and "active" without the aggression of red.
        slate: {
          950: "#0a0f1e",
          900: "#0f172a",
          850: "#131c35",
          800: "#1e293b",
          700: "#334155",
          600: "#475569",
          400: "#94a3b8",
          200: "#e2e8f0",
          100: "#f1f5f9",
        },
        amber: {
          500: "#f59e0b",
          400: "#fbbf24",
          300: "#fcd34d",
        },
        emerald: {
          500: "#10b981",
          400: "#34d399",
        },
        rose: {
          500: "#f43f5e",
          400: "#fb7185",
        },
      },
      animation: {
        "fade-in": "fadeIn 0.3s ease-out",
        "slide-up": "slideUp 0.4s ease-out",
      },
      keyframes: {
        fadeIn: {
          "0%": { opacity: "0" },
          "100%": { opacity: "1" },
        },
        slideUp: {
          "0%": { opacity: "0", transform: "translateY(8px)" },
          "100%": { opacity: "1", transform: "translateY(0)" },
        },
      },
    },
  },
  plugins: [],
};

export default config;
