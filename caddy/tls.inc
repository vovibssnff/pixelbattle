# TLS overrides for the main site (imported inside the {$CADDY_DOMAIN} block).
#
# Default for production: Caddy obtains public certificates via Let's Encrypt
# (needs CADDY_DOMAIN with DNS pointing here, ACME_EMAIL, ports 80/443 reachable).
#
# Local / lab without public DNS — uncomment:
#	tls internal
#
# Caddy-managed internal CA — no cert files needed (bench/lab mode):
	tls internal
