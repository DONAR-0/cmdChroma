import { defineConfig } from 'vite';
import tailwindcss from '@tailwindcss/vite';
import { TanStackRouterVite } from '@tanstack/router-devtools/vite';

export default defineConfig({
  plugins: [
    tailwindcss(),
    TanStackRouterVite(),
  ],
  server: {
    port: 3000,
    proxy: {
      '/api': {
        target: process.env.VITE_API_URL || 'http://localhost:6700',
        changeOrigin: true,
      },
      '/health': {
        target: process.env.VITE_API_URL || 'http://localhost:6700',
        changeOrigin: true,
      },
    },
  },
});