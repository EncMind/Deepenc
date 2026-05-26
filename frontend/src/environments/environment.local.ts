export const environment = {
  production: false,
  apiUrl: '/api',
  firebase: null,
  debugMode: true,
  enableHotReload: true,
  devtools: true,
  verbose: true,
  useDynamicConfig: true,
  configEndpoints: {
    firebase: '/api/config/firebase',
    stripe: '/api/config/stripe'
  },
  backend: {
    host: '172.200.99.119',
    port: 8080,
    protocol: 'http'
  }
};