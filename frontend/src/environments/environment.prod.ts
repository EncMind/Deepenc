export const environment = {
  production: true,
  apiUrl: '/api',
  firebase: null,
  debugMode: false,
  enableHotReload: false,
  useDynamicConfig: true,
  configEndpoints: {
    firebase: '/api/config/firebase',
    stripe: '/api/config/stripe'
  },
  stripePublishableKey: null as string | null
};