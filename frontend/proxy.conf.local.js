const PROXY_CONFIG = [
  {
    context: ['/api/**'],
    target: "http://172.200.99.119:8080",
    secure: false,
    changeOrigin: true,
    logLevel: "debug",
    onProxyReq: function(proxyReq, req, res) {
      console.log('\n[PROXY] Request:', req.method, req.url);
      console.log('[PROXY] Target:', "http://172.200.99.119:8080" + req.url);
      // Sanitize sensitive headers before logging
      const headers = { ...req.headers };
      if (headers.authorization) headers.authorization = 'Bearer ***';
      if (headers.cookie) headers.cookie = '***';
      console.log('[PROXY] Headers:', JSON.stringify(headers, null, 2));
    },
    onProxyRes: function(proxyRes, req, res) {
      console.log('[PROXY] Response:', proxyRes.statusCode, proxyRes.statusMessage);
      console.log('[PROXY] Response Headers:', JSON.stringify(proxyRes.headers, null, 2));
    },
    onError: function(err, req, res) {
      console.log('[PROXY] Error:', err.message);
      console.log('[PROXY] Error details:', err);
    }
  }
];

module.exports = PROXY_CONFIG;
