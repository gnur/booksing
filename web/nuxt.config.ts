// https://nuxt.com/docs/api/configuration/nuxt-config
export default defineNuxtConfig({
  devtools: { enabled: true },

  nitro: {
    devProxy: {
      "/api": {
        target: "http://127.0.0.1:7133/api",
        changeOrigin: false,
        prependPath: true,
        toProxy: true,
      }
    }
  },

  modules: ['@nuxt/ui'],

  ssr: false
})
