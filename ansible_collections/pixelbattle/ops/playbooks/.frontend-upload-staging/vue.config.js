const { defineConfig } = require('@vue/cli-service')
module.exports = defineConfig({
  transpileDependencies: true,
  devServer: {
    server: 'https',
    allowedHosts: "all",
    hot: false,
    liveReload: false,
  },
  // chainWebpack: (config) => {
  //   config.resolve.symlinks(false)
  // }
})
