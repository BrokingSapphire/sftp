FROM nginx:1.27-alpine
# Bake the reverse-proxy config into the image so no host bind-mount is needed.
COPY nginx.conf /etc/nginx/conf.d/default.conf
# To enable TLS, COPY your certs here and uncomment the 443 block in nginx.conf:
# COPY certs/ /etc/nginx/certs/
