module.exports = {
  devServer: {
    proxy: {
      '^/api': {
        target: 'http://localhost:7132'
      },
      '^/auth': {
        target: 'http://localhost:7132'
      }
    }
  }
}
