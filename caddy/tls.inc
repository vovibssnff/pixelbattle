# Optional TLS overrides for the main site (imported inside the {$CADDY_DOMAIN} block).
#
# Default — leave as comments only: Caddy obtains public certificates via Let's Encrypt
# (needs CADDY_DOMAIN with DNS pointing here, ACME_EMAIL, ports 80/443 reachable from the internet).
#
# Local / lab without public DNS — uncomment:
#	tls internal
#
# Your own PEMs in ./certs (e.g. on the VM before DNS/ACME is ready) — uncomment ONE pair:
#	tls /etc/caddy/certs/fullchain.pem /etc/caddy/certs/privkey.pem
#	tls /etc/caddy/certs/pixelbattle.crt /etc/caddy/certs/pixelbattle.key
