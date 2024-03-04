// https://nuxt.com/docs/api/configuration/nuxt-config
export default defineNuxtConfig({
  devtools: { enabled: true },

  nitro: {
    devProxy: {
      "/api/search": {
        target: "http://127.0.0.1:7133/api/search",
        changeOrigin: false,
        prependPath: true,
        toProxy: true,
      },
      "/download": {
        target: "http://127.0.0.1:7133/download",
        changeOrigin: false,
        prependPath: true,
        toProxy: true,
      },
      "/cover": {
        target: "http://127.0.0.1:7133/cover",
        changeOrigin: false,
        prependPath: true,
        toProxy: true,
      }
    }
  },

  modules: ['@nuxt/ui'],

  ssr: false
})
