# TLS overrides for the main site (imported inside the {$CADDY_DOMAIN} block).
#
# Default for production: Caddy obtains public certificates via Let's Encrypt
# (needs CADDY_DOMAIN with DNS pointing here, ACME_EMAIL, ports 80/443 reachable).
#
# Local / lab without public DNS — uncomment:
#	tls internal
#
# Self-signed PEMs for bench/dev (SANs: pixelbattle.local, localhost, 192.168.122.71):
	tls /etc/caddy/certs/pixelbattle.crt /etc/caddy/certs/pixelbattle.key
