# Changelog

## [0.1.2](https://github.com/alleato-llc/dnsctl/compare/v0.1.1...v0.1.2) (2026-06-03)


### Features

* **gui:** add a Font setting (System / Rounded / Mono) with live previews ([8812048](https://github.com/alleato-llc/dnsctl/commit/8812048e141afc492930d0bd2b3a26e5414c142c))
* **gui:** add helper-status, system-hosts view, and confirm-before-remove ([73a9769](https://github.com/alleato-llc/dnsctl/commit/73a97694111164ac294f68a5a635ca4c8d2294be))
* **gui:** add Settings with Light/Dark/System theme (gear icon) ([ae9513e](https://github.com/alleato-llc/dnsctl/commit/ae9513eaa1de3fbf9145a2a75df494315763bf86))
* **gui:** flag the active (default-route) service in DNS Status ([53dc6f3](https://github.com/alleato-llc/dnsctl/commit/53dc6f3a9e16575b6fca9fb316fd749568ceac18))
* **gui:** modern macOS-style UI with a read-only DNS Status view ([09ba3d8](https://github.com/alleato-llc/dnsctl/commit/09ba3d82c002aea42bcab23d7225a229c97be4d3))
* **gui:** wire a Wails frontend onto the service seam via guiapi ([a2204a3](https://github.com/alleato-llc/dnsctl/commit/a2204a369353b3f5f1b73c324ff6bb3ce0612c74))
* **helper:** add privileged helper daemon and wire HelperClient over IPC ([c6b5d5f](https://github.com/alleato-llc/dnsctl/commit/c6b5d5f745bc5e80b227ecad4bb6d0884e4b5383))
* **helper:** production wiring — peer-UID auth, auto-routing, launchd packaging ([1a4f6a4](https://github.com/alleato-llc/dnsctl/commit/1a4f6a4ad80ac005795d36578aa31e2d9353f0ce))
* **hosts:** add headless /etc/hosts CRUD via cobra subcommands ([0679ab2](https://github.com/alleato-llc/dnsctl/commit/0679ab231aea9abe120fd3d77d9162a0f729c7ad))
* manage DNS profiles and edit resolver config across CLI and GUI ([7ad01f4](https://github.com/alleato-llc/dnsctl/commit/7ad01f40ac8cf3cecc77856d5ec511762423d9d6))


### Bug Fixes

* **gui:** expose App.Backend() so the binding (and DNS view) resolves ([028271f](https://github.com/alleato-llc/dnsctl/commit/028271f02752ed3cca17e07abac6db6e01dd6bde))
* **gui:** make the Wails module build out of the box; doc copy-paste fixes ([b85748d](https://github.com/alleato-llc/dnsctl/commit/b85748d9e146feff70c69478ac6cc264d1e3514f))
* **helper:** make the socket connectable by non-root clients (0666) ([014c767](https://github.com/alleato-llc/dnsctl/commit/014c767b06da52b2dac036f7c08a3e980f6fe502))

## [0.1.1](https://github.com/nycjv321/dnsctl/compare/v0.1.0...v0.1.1) (2026-01-20)


### Features

* **ci:** add CI/CD pipeline and release automation ([77fd307](https://github.com/nycjv321/dnsctl/commit/77fd3075d7b4a212796e0c506798ddb4d4b70996))


### Bug Fixes

* **ci:** add release-please config files for v4 compatibility ([df88ebc](https://github.com/nycjv321/dnsctl/commit/df88ebc3e702f08da6179f6894cb2b4d73ddbcdf))
* **ci:** resolve CI/CD pipeline issues ([48525ce](https://github.com/nycjv321/dnsctl/commit/48525cef3d16b7c602ee753236c42f98f9800ac1))
* **config:** return "auto" as default service on Linux ([8a54f59](https://github.com/nycjv321/dnsctl/commit/8a54f5914c9993a56c6f1fd03a5cabfdbc519348))
