import { defineConfig } from 'vite';
import tailwindcss from '@tailwindcss/vite';
import router from '@tanstack/router-plugin/vite';

export default defineConfig({
  plugins: [tailwindcss(), router()],
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