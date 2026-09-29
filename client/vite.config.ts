import react, { reactCompilerPreset } from '@vitejs/plugin-react'
import babel from '@rolldown/plugin-babel'
import tailwindcss from '@tailwindcss/vite'
import { defineConfig } from 'vite'

export default defineConfig({
  base: '/vaultguard/',
  plugins: [
    tailwindcss(),
    react(),
    babel({ presets: [reactCompilerPreset()] })
  ],
  resolve: {
    alias: {
      '@': `${import.meta.dirname}/src`,
    },
  },
  server: {
    port: 7540,
    host: true,
    proxy: {
      // Dev-time reverse proxy so the relative VITE_API_URL (/vaultguard/api)
      // reaches the Go server at http://localhost:7541 without CORS issues.
      // Mirrors the nginx.conf proxy used in docker.
      '/vaultguard/api': {
        target: 'http://localhost:7541',
        changeOrigin: true,
        ws: true,
      },
    },
  }
})
