export const environment = {
  production: false,
  apiUrl: '/api',
  firebase: null,
  stripePublishableKey: null,
  debugMode: true,
  enableHotReload: true,
  useDynamicConfig: true,
  configEndpoints: {
    firebase: '/api/config/firebase',
    stripe: '/api/config/stripe'
  }
};