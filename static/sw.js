const CACHE_NAME = 'alumni-tracker-v2';
const ASSETS_TO_CACHE = [
  '/',
  '/info',
  '/static/icon-192.png',
  '/static/icon-512.png'
];

// Install event: cache initial shell assets
self.addEventListener('install', (event) => {
  event.waitUntil(
    caches.open(CACHE_NAME).then((cache) => {
      return cache.addAll(ASSETS_TO_CACHE);
    })
  );
  self.skipWaiting();
});

// Activate event: clean old caches
self.addEventListener('activate', (event) => {
  event.waitUntil(
    caches.keys().then((keys) => {
      return Promise.all(
        keys.map((key) => {
          if (key !== CACHE_NAME) {
            return caches.delete(key);
          }
        })
      );
    })
  );
  self.clients.claim();
});

// Fetch event: Network-first strategy with cache fallback
self.addEventListener('fetch', (event) => {
  // Only handle HTTP/HTTPS
  if (!event.request.url.startsWith(self.location.origin)) return;

  // Bypass cache for non-GET requests
  if (event.request.method !== 'GET') {
    return;
  }

  // Never cache manifest.json or API calls — always go to network
  const url = new URL(event.request.url);
  if (url.pathname === '/manifest.json' || url.pathname.startsWith('/api/')) {
    event.respondWith(fetch(event.request));
    return;
  }

  event.respondWith(
    fetch(event.request)
      .then((networkResponse) => {
        // If response is valid, update the cache copy
        if (networkResponse && networkResponse.status === 200) {
          const responseToCache = networkResponse.clone();
          caches.open(CACHE_NAME).then((cache) => {
            cache.put(event.request, responseToCache);
          });
        }
        return networkResponse;
      })
      .catch(() => {
        // Network failed, try cache
        return caches.match(event.request).then((cachedResponse) => {
          if (cachedResponse) {
            return cachedResponse;
          }
          // If completely offline and page doesn't exist, try returning root
          if (event.request.headers.get('accept') && event.request.headers.get('accept').includes('text/html')) {
            return caches.match('/');
          }
        });
      })
  );
});
