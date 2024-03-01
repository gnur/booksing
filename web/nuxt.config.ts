// https://nuxt.com/docs/api/configuration/nuxt-config
export default defineNuxtConfig({
  devtools: { enabled: true },

  nitro: {
    devProxy: {
      "/api/search": {
        target: "http://127.0.0.1:7133/api/search",
        changeOrigin: false,
        prependPath: true,
      }
    }
  },

  modules: ['@nuxt/ui'],

  ssr: false
})
