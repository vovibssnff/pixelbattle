# TLS overrides for the main site (imported inside the {$CADDY_DOMAIN} block).
#
# Production default: leave this file without a `tls` directive so Caddy obtains
# public certificates via ACME/Let's Encrypt. This requires CADDY_DOMAIN DNS to
# point at the Caddy node and public ports 80/443 to be reachable.
#
# For local/lab mode without public DNS, use:
#   tls internal
#
# For self-signed PEMs, generate files under /opt/pixelbattle/certs on the
# manager, bind-mount them into the caddy service, and use:
#   tls /etc/caddy/certs/pixelbattle.crt /etc/caddy/certs/pixelbattle.key
