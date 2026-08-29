# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.19](https://github.com/coolify-terraform/terraform-provider-coolify/compare/v0.1.18...v0.1.19) (2026-08-29)


### Features

* pin Coolify v4.3.14 and prove list-order flatten ([#821](https://github.com/coolify-terraform/terraform-provider-coolify/issues/821)) ([2ec35d3](https://github.com/coolify-terraform/terraform-provider-coolify/commit/2ec35d3eab3f7a40728de9eee494425b6f830588))


### Bug Fixes

* **application:** keep noindex_domains configured order ([#820](https://github.com/coolify-terraform/terraform-provider-coolify/issues/820)) ([114f480](https://github.com/coolify-terraform/terraform-provider-coolify/commit/114f480d5340df15ed798134770631b391472569)), closes [#818](https://github.com/coolify-terraform/terraform-provider-coolify/issues/818)
* **ci:** upsert one social preview reminder issue ([#816](https://github.com/coolify-terraform/terraform-provider-coolify/issues/816)) ([a8dedc5](https://github.com/coolify-terraform/terraform-provider-coolify/commit/a8dedc595c4966a160e15301475ce6a71cc694f8)), closes [#774](https://github.com/coolify-terraform/terraform-provider-coolify/issues/774)
* extra-key probe smtp_ehlo_domain on CI edge ([#814](https://github.com/coolify-terraform/terraform-provider-coolify/issues/814)) ([3bcdf6b](https://github.com/coolify-terraform/terraform-provider-coolify/commit/3bcdf6b164ac1076865c52b31f7648834790eb7e))
* **service:** preserve configured urls order on read-back ([#819](https://github.com/coolify-terraform/terraform-provider-coolify/issues/819)) ([d2c3b30](https://github.com/coolify-terraform/terraform-provider-coolify/commit/d2c3b300746c8ec637702be1acfdbcfeba5bcd49)), closes [#818](https://github.com/coolify-terraform/terraform-provider-coolify/issues/818)
* wait for social preview confirm before leaving Settings ([#817](https://github.com/coolify-terraform/terraform-provider-coolify/issues/817)) ([864c709](https://github.com/coolify-terraform/terraform-provider-coolify/commit/864c709ec554c83802ed7617548fdce748e8c000))

## [0.1.18](https://github.com/coolify-terraform/terraform-provider-coolify/compare/v0.1.17...v0.1.18) (2026-08-22)


### Features

* add instance email settings data source ([#804](https://github.com/coolify-terraform/terraform-provider-coolify/issues/804)) ([4632a5a](https://github.com/coolify-terraform/terraform-provider-coolify/commit/4632a5a407a36372acad904b082ad2152b0d3bd1))
* **notification:** add smtp_ehlo_domain to email notifications ([#800](https://github.com/coolify-terraform/terraform-provider-coolify/issues/800)) ([f49d954](https://github.com/coolify-terraform/terraform-provider-coolify/commit/f49d954f238e46fdd98855f86b602f89a9a3e74e))
* pin Coolify v4.3.10 and add instance email settings ([#802](https://github.com/coolify-terraform/terraform-provider-coolify/issues/802)) ([6346ad7](https://github.com/coolify-terraform/terraform-provider-coolify/commit/6346ad7f02520dacc94f972fb3ed8864101fc946))


### Bug Fixes

* validate SMTP from-address and extract instance email FIELDS ([#805](https://github.com/coolify-terraform/terraform-provider-coolify/issues/805)) ([23768ac](https://github.com/coolify-terraform/terraform-provider-coolify/commit/23768ac5182dc8ae168b0eeb78ad594b357b31dd))

## [0.1.17](https://github.com/coolify-terraform/terraform-provider-coolify/compare/v0.1.16...v0.1.17) (2026-08-19)


### Features

* pin Coolify API contract to v4.3.9 ([#792](https://github.com/coolify-terraform/terraform-provider-coolify/issues/792)) ([9e906d6](https://github.com/coolify-terraform/terraform-provider-coolify/commit/9e906d665b79d8ecf1067549fcf645033ff5498c)), closes [#776](https://github.com/coolify-terraform/terraform-provider-coolify/issues/776)


### Bug Fixes

* **application:** keep explicit docker image tag on update ([#788](https://github.com/coolify-terraform/terraform-provider-coolify/issues/788)) ([4043caf](https://github.com/coolify-terraform/terraform-provider-coolify/commit/4043cafba6219805f72fd5fbf201a743263f8cac))
* **application:** split docker image into name and tag on update ([#784](https://github.com/coolify-terraform/terraform-provider-coolify/issues/784)) ([7dd47f7](https://github.com/coolify-terraform/terraform-provider-coolify/commit/7dd47f704fb80b6da3c8d308c14d3ad66de4ff0a))
* **database:** omit Coolify-rejected fields from create PATCH ([#790](https://github.com/coolify-terraform/terraform-provider-coolify/issues/790)) ([2a0b5a6](https://github.com/coolify-terraform/terraform-provider-coolify/commit/2a0b5a6a32a5178a3c3f1b88bc0de5c4ccd96d83)), closes [#789](https://github.com/coolify-terraform/terraform-provider-coolify/issues/789)
* **deployment:** deploy on first apply instead of restart ([#785](https://github.com/coolify-terraform/terraform-provider-coolify/issues/785)) ([fd0208f](https://github.com/coolify-terraform/terraform-provider-coolify/commit/fd0208fdf3af968afaececcbabe807ee225a8a4a))
* union Coolify allowlists and cover service PATCH ([#795](https://github.com/coolify-terraform/terraform-provider-coolify/issues/795)) ([2d01c95](https://github.com/coolify-terraform/terraform-provider-coolify/commit/2d01c95952fedc95dbda592c76e165d782efc4cb))

## [0.1.16](https://github.com/coolify-terraform/terraform-provider-coolify/compare/v0.1.15...v0.1.16) (2026-08-16)


### Features

* pin Coolify API contract to v4.3.3 ([#777](https://github.com/coolify-terraform/terraform-provider-coolify/issues/777)) ([ee3bc76](https://github.com/coolify-terraform/terraform-provider-coolify/commit/ee3bc76b3a8b1b81f94f37f86116167b951cc022))
* pin Coolify API contract to v4.3.5 ([#779](https://github.com/coolify-terraform/terraform-provider-coolify/issues/779)) ([6160fac](https://github.com/coolify-terraform/terraform-provider-coolify/commit/6160face0e3582383c2d32a6bb260cce59958645))


### Bug Fixes

* persist server proxy redirect_url when type is unchanged ([#780](https://github.com/coolify-terraform/terraform-provider-coolify/issues/780)) ([ace788e](https://github.com/coolify-terraform/terraform-provider-coolify/commit/ace788e0590dcfbba68ada4b04d9d727d23cb669))

## [0.1.15](https://github.com/coolify-terraform/terraform-provider-coolify/compare/v0.1.14...v0.1.15) (2026-08-14)


### Features

* **application:** consistent container name docs and API parity acc depth ([#756](https://github.com/coolify-terraform/terraform-provider-coolify/issues/756)) ([005cb91](https://github.com/coolify-terraform/terraform-provider-coolify/commit/005cb9129df2e1839d594d8d576229883fed75dd))
* Coolify API parity for GitLab Apps, tags, shared envs, and server control ([#752](https://github.com/coolify-terraform/terraform-provider-coolify/issues/752)) ([f7bf746](https://github.com/coolify-terraform/terraform-provider-coolify/commit/f7bf7467997d49c273db45d221b1d06a75b89e51))
* enable Hetzner backups and cover DO/Vultr list clients ([#767](https://github.com/coolify-terraform/terraform-provider-coolify/issues/767)) ([ee666bd](https://github.com/coolify-terraform/terraform-provider-coolify/commit/ee666bdd451da02632b3a690f1fdac5608b8c2c0)), closes [#764](https://github.com/coolify-terraform/terraform-provider-coolify/issues/764) [#765](https://github.com/coolify-terraform/terraform-provider-coolify/issues/765)
* expose Coolify 4.3.3 tip GET fields ([#758](https://github.com/coolify-terraform/terraform-provider-coolify/issues/758)) ([1427672](https://github.com/coolify-terraform/terraform-provider-coolify/commit/142767269efbdacc511b89a9fee670afcb0e632b)), closes [#719](https://github.com/coolify-terraform/terraform-provider-coolify/issues/719)
* **hetzner:** attach networks and firewalls on coolify_server_hetzner ([#761](https://github.com/coolify-terraform/terraform-provider-coolify/issues/761)) ([bc1dd32](https://github.com/coolify-terraform/terraform-provider-coolify/commit/bc1dd324f9360c41923babb9d6704c1ee0472f20)), closes [#760](https://github.com/coolify-terraform/terraform-provider-coolify/issues/760)


### Bug Fixes

* floor acc skips, shared env key validation, and docs counts ([#757](https://github.com/coolify-terraform/terraform-provider-coolify/issues/757)) ([4455b56](https://github.com/coolify-terraform/terraform-provider-coolify/commit/4455b56b4bfb9c05ff96d983b5ec2cca20d19db8))
* redact API keys in logs and cover server control-plane ([#769](https://github.com/coolify-terraform/terraform-provider-coolify/issues/769)) ([06fdb83](https://github.com/coolify-terraform/terraform-provider-coolify/commit/06fdb839a0d0cca8edd764556fed97281fde1b6a))
* send Hetzner SSH key IDs as JSON arrays ([#766](https://github.com/coolify-terraform/terraform-provider-coolify/issues/766)) ([eb7198f](https://github.com/coolify-terraform/terraform-provider-coolify/commit/eb7198f2451fcc2941e7487218df8653bf1ca6ba))
* stop server proxy acc from setting redirect_enabled false ([#768](https://github.com/coolify-terraform/terraform-provider-coolify/issues/768)) ([c9e743d](https://github.com/coolify-terraform/terraform-provider-coolify/commit/c9e743d1202538c736ba35f22af703033aef501a))

## [0.1.14](https://github.com/coolify-terraform/terraform-provider-coolify/compare/v0.1.13...v0.1.14) (2026-08-13)


### Features

* coolify_notification_discord and coolify_notification_slack ([#703](https://github.com/coolify-terraform/terraform-provider-coolify/issues/703)) ([46f8388](https://github.com/coolify-terraform/terraform-provider-coolify/commit/46f838871559195d862de7c045a7a1d0c1b846f6))
* coolify_notification_email and coolify_notification_telegram ([#706](https://github.com/coolify-terraform/terraform-provider-coolify/issues/706)) ([cf398bb](https://github.com/coolify-terraform/terraform-provider-coolify/commit/cf398bbac71d781882e5cd23cb85e16636425cc9))
* coolify_notification_webhook and coolify_notification_pushover ([#705](https://github.com/coolify-terraform/terraform-provider-coolify/issues/705)) ([fb0d7ad](https://github.com/coolify-terraform/terraform-provider-coolify/commit/fb0d7add2d8c4ca64a80e85402c8a9bab6845239))
* coolify_s3_storage and CI OpenTofu flake hardening ([#689](https://github.com/coolify-terraform/terraform-provider-coolify/issues/689)) ([23c5952](https://github.com/coolify-terraform/terraform-provider-coolify/commit/23c5952644c16fe7ec5e68869fd571fde0d5c5c8))
* coolify_s3_storage_validate and shared create read-back helpers ([#698](https://github.com/coolify-terraform/terraform-provider-coolify/issues/698)) ([f333989](https://github.com/coolify-terraform/terraform-provider-coolify/commit/f33398957cd2e3e189689abc7e1cba1a2fffd920))
* drive API_COVERAGE from Coolify contract routes ([#702](https://github.com/coolify-terraform/terraform-provider-coolify/issues/702)) ([b250364](https://github.com/coolify-terraform/terraform-provider-coolify/commit/b2503646171bad4629c38e169b54ae43db1f0349))
* expose Coolify tip docker/compose version probes on servers ([#708](https://github.com/coolify-terraform/terraform-provider-coolify/issues/708)) ([c8a1699](https://github.com/coolify-terraform/terraform-provider-coolify/commit/c8a16995963627429cd468e4ce0069da2fd92e3f))
* notification data sources, 404 tests, and event mapping errors ([#725](https://github.com/coolify-terraform/terraform-provider-coolify/issues/725)) ([2d4e4e1](https://github.com/coolify-terraform/terraform-provider-coolify/commit/2d4e4e1e3fe676f1623bf69f3c902a17d7d5e3f3))
* pin Coolify API contract to v4.3.2 ([#709](https://github.com/coolify-terraform/terraform-provider-coolify/issues/709)) ([d791afa](https://github.com/coolify-terraform/terraform-provider-coolify/commit/d791afa79c78096d81ef7a0606490c3840604472)), closes [#699](https://github.com/coolify-terraform/terraform-provider-coolify/issues/699)
* pin Coolify contract to v4.3.1 and watch tip API early ([#686](https://github.com/coolify-terraform/terraform-provider-coolify/issues/686)) ([8da70af](https://github.com/coolify-terraform/terraform-provider-coolify/commit/8da70af0c04f49fc08dc6c02cf990eed3343d561))
* shared notification helpers and acme-notifications scenario ([#715](https://github.com/coolify-terraform/terraform-provider-coolify/issues/715)) ([fac7535](https://github.com/coolify-terraform/terraform-provider-coolify/commit/fac7535ccf32062b411fa2b464b2829b605e2292))


### Bug Fixes

* align notification mapping errors and thread tests ([#729](https://github.com/coolify-terraform/terraform-provider-coolify/issues/729)) ([fe28850](https://github.com/coolify-terraform/terraform-provider-coolify/commit/fe28850a3f56eb95b7ab312694c61cb8a712e78d))
* bump Go 1.26.5 to 1.26.6 for stdlib CVEs ([#734](https://github.com/coolify-terraform/terraform-provider-coolify/issues/734)) ([2ec06ce](https://github.com/coolify-terraform/terraform-provider-coolify/commit/2ec06cec2156b5a72a2d88ebff0709631f3f0a78))
* include Coolify deploy logs and retry acme-github-cicd ([#730](https://github.com/coolify-terraform/terraform-provider-coolify/issues/730)) ([8a2df24](https://github.com/coolify-terraform/terraform-provider-coolify/commit/8a2df2485c2f1f023308081cd88b434645993bdf)), closes [#728](https://github.com/coolify-terraform/terraform-provider-coolify/issues/728)
* S3 docs accuracy, CI scenario path filter, DS error IDs ([#694](https://github.com/coolify-terraform/terraform-provider-coolify/issues/694)) ([336b046](https://github.com/coolify-terraform/terraform-provider-coolify/commit/336b046cd2906204716ad021b26ed3f89b2bbf72))

## [0.1.13](https://github.com/coolify-terraform/terraform-provider-coolify/compare/v0.1.12...v0.1.13) (2026-08-10)


### Features

* expose destination_uuid on applications and databases ([#678](https://github.com/coolify-terraform/terraform-provider-coolify/issues/678)) ([ee533a0](https://github.com/coolify-terraform/terraform-provider-coolify/commit/ee533a07e6e6e343aac9d723ab40f56d745ea72e))
* **service:** expose destination_uuid on coolify_service ([#680](https://github.com/coolify-terraform/terraform-provider-coolify/issues/680)) ([3da5208](https://github.com/coolify-terraform/terraform-provider-coolify/commit/3da5208682f02e54dda5dc17b5954d955afbad5a))

## [0.1.12](https://github.com/coolify-terraform/terraform-provider-coolify/compare/v0.1.11...v0.1.12) (2026-08-05)


### Features

* **application:** warn when 4.2-only settings set on Coolify &lt; 4.2.0 ([#664](https://github.com/coolify-terraform/terraform-provider-coolify/issues/664)) ([50f657e](https://github.com/coolify-terraform/terraform-provider-coolify/commit/50f657eae618bcd63a5bd9f2fb10da907f467d5c)), closes [#663](https://github.com/coolify-terraform/terraform-provider-coolify/issues/663)


### Bug Fixes

* **application:** normalize docker_compose_domains array vs object ([#658](https://github.com/coolify-terraform/terraform-provider-coolify/issues/658)) ([90bfe97](https://github.com/coolify-terraform/terraform-provider-coolify/commit/90bfe976a623523274251e927892b3fa74a7414f))
* **contract:** expand PHP ...self::CONST spreads in allow-list extract ([#665](https://github.com/coolify-terraform/terraform-provider-coolify/issues/665)) ([d280f37](https://github.com/coolify-terraform/terraform-provider-coolify/commit/d280f378f9298a2b146f81737e9ba3c4219c8ba5)), closes [#661](https://github.com/coolify-terraform/terraform-provider-coolify/issues/661)
* **security:** scorecard improvements (vulns, provenance, branch) ([#669](https://github.com/coolify-terraform/terraform-provider-coolify/issues/669)) ([3677e33](https://github.com/coolify-terraform/terraform-provider-coolify/commit/3677e33e43c203f96d3a9b1f96deff7a55cbd895))
* version-gate Coolify 4.2-only application write fields ([#662](https://github.com/coolify-terraform/terraform-provider-coolify/issues/662)) ([91814a5](https://github.com/coolify-terraform/terraform-provider-coolify/commit/91814a59095a458d4871f471c5ab923386cd44e6))

## [0.1.11](https://github.com/coolify-terraform/terraform-provider-coolify/compare/v0.1.10...v0.1.11) (2026-08-01)


### Features

* clear deferred contract field umbrella ([#626](https://github.com/coolify-terraform/terraform-provider-coolify/issues/626)) ([#641](https://github.com/coolify-terraform/terraform-provider-coolify/issues/641)) ([ac8ebdc](https://github.com/coolify-terraform/terraform-provider-coolify/commit/ac8ebdcc2e60534997d0713446c5ca85cac070e3))
* expose autogenerate_domain and related domain controls ([#646](https://github.com/coolify-terraform/terraform-provider-coolify/issues/646)) ([802a9fc](https://github.com/coolify-terraform/terraform-provider-coolify/commit/802a9fcd740fe561c978224b9896d6ae8b22a410))


### Bug Fixes

* **ci:** drop em dash from social-preview repo description ([#639](https://github.com/coolify-terraform/terraform-provider-coolify/issues/639)) ([36fd7ae](https://github.com/coolify-terraform/terraform-provider-coolify/commit/36fd7ae78b35fc038a84105c5a4435972e73b1d3))

## [0.1.10](https://github.com/coolify-terraform/terraform-provider-coolify/compare/v0.1.9...v0.1.10) (2026-07-30)


### Features

* application preview/build secrets/stop grace and scheduled task container/timeout ([#630](https://github.com/coolify-terraform/terraform-provider-coolify/issues/630)) ([7581335](https://github.com/coolify-terraform/terraform-provider-coolify/commit/7581335057c2e3ac962ce86678f5a3969810f40a))
* expose env is_runtime, is_literal, is_multiline, and comment ([#625](https://github.com/coolify-terraform/terraform-provider-coolify/issues/625)) ([a5dad15](https://github.com/coolify-terraform/terraform-provider-coolify/commit/a5dad15772401d01f61ea302a8b4bcb122a76046))
* github_app SSH fields plus MPI test and docs polish ([#633](https://github.com/coolify-terraform/terraform-provider-coolify/issues/633)) ([fb4746a](https://github.com/coolify-terraform/terraform-provider-coolify/commit/fb4746a16fe2aa2447bfff94754d3bc92319a30d))


### Bug Fixes

* accept legacy short Coolify identifiers in UUID validation ([8009c61](https://github.com/coolify-terraform/terraform-provider-coolify/commit/8009c61cf184176b38a4e3a78778d739297db72c))
* accept unknown docker_compose_raw/type in service ValidateConfig ([#618](https://github.com/coolify-terraform/terraform-provider-coolify/issues/618)) ([4ad9ae0](https://github.com/coolify-terraform/terraform-provider-coolify/commit/4ad9ae014c96e049958d9b88b360556582b79cbc))
* **ci:** skip Auto Approve on fork PRs ([#613](https://github.com/coolify-terraform/terraform-provider-coolify/issues/613)) ([fff6810](https://github.com/coolify-terraform/terraform-provider-coolify/commit/fff68104e475ec5c43d64a3f96d17afbaa7efea5))
* richer API error context and Coolify id troubleshooting ([#617](https://github.com/coolify-terraform/terraform-provider-coolify/issues/617)) ([e54443c](https://github.com/coolify-terraform/terraform-provider-coolify/commit/e54443ce4d4f0ff4dcd1a24f5dbd46f9e3f00d94))
* ValidateConfig unknown guards for backups and richer client errors ([#620](https://github.com/coolify-terraform/terraform-provider-coolify/issues/620)) ([c37c0af](https://github.com/coolify-terraform/terraform-provider-coolify/commit/c37c0af8221e248eebf6fa3d385fe8e46a7d14fc))

## [0.1.9](https://github.com/coolify-terraform/terraform-provider-coolify/compare/v0.1.8...v0.1.9) (2026-07-27)


### Features

* Coolify v4.2 contract, DigitalOcean/Vultr servers, destinations ([#589](https://github.com/coolify-terraform/terraform-provider-coolify/issues/589)) ([6ae4681](https://github.com/coolify-terraform/terraform-provider-coolify/commit/6ae4681243514bc8525cba944e08bc6fe379404c))
* coolify_storage_backup for volume backup schedules ([#601](https://github.com/coolify-terraform/terraform-provider-coolify/issues/601)) ([e5a9ed5](https://github.com/coolify-terraform/terraform-provider-coolify/commit/e5a9ed56c48f00a5dba2665af344fd72b2be757e))


### Bug Fixes

* accept Coolify bare human cron schedules ([#603](https://github.com/coolify-terraform/terraform-provider-coolify/issues/603)) ([e0f641f](https://github.com/coolify-terraform/terraform-provider-coolify/commit/e0f641f71afe2dbc08b399bd4cda46c03c71a063))
* destination coverage, empty UUID guards, and v4.2 docs ([#598](https://github.com/coolify-terraform/terraform-provider-coolify/issues/598)) ([d184aba](https://github.com/coolify-terraform/terraform-provider-coolify/commit/d184aba0ab7639489083a0ba5b6eb971732f0ed8))
* preserve raw custom_nginx_configuration on read ([#604](https://github.com/coolify-terraform/terraform-provider-coolify/issues/604)) ([7e3a33b](https://github.com/coolify-terraform/terraform-provider-coolify/commit/7e3a33b807c7911ac7ad7692309197b5a282aff7))
* use POST for Coolify action/validate endpoints ([#587](https://github.com/coolify-terraform/terraform-provider-coolify/issues/587)) ([e03c47f](https://github.com/coolify-terraform/terraform-provider-coolify/commit/e03c47f47be0a6c3df3d786d107217d38d98c003))

## [0.1.8](https://github.com/coolify-terraform/terraform-provider-coolify/compare/v0.1.7...v0.1.8) (2026-07-18)


### Bug Fixes

* application webhook secrets, import safety, seed fields, docs ([#579](https://github.com/coolify-terraform/terraform-provider-coolify/issues/579)) ([6d03dc3](https://github.com/coolify-terraform/terraform-provider-coolify/commit/6d03dc32eddc708b8d0c8677eca71ae884aea3d0))
* bump Go 1.26.4 to 1.26.5 (GO-2026-5856) ([#570](https://github.com/coolify-terraform/terraform-provider-coolify/issues/570)) ([3334377](https://github.com/coolify-terraform/terraform-provider-coolify/commit/3334377f65bf98a43b87f6395441d9b944455ba1))
* compound import server validation for databases and services ([#580](https://github.com/coolify-terraform/terraform-provider-coolify/issues/580)) ([6b20719](https://github.com/coolify-terraform/terraform-provider-coolify/commit/6b20719a653a54e144ad727578cd059ad4cd0446))

## [0.1.7](https://github.com/coolify-terraform/terraform-provider-coolify/compare/v0.1.6...v0.1.7) (2026-06-26)


### Bug Fixes

* **github_app:** import by app_id instead of internal id ([#559](https://github.com/coolify-terraform/terraform-provider-coolify/issues/559)) ([7250183](https://github.com/coolify-terraform/terraform-provider-coolify/commit/72501832b18ee87f8105108d9bf22702668f705a))
* multi-perspective improvement rotation (client tests, doc fixes) ([#561](https://github.com/coolify-terraform/terraform-provider-coolify/issues/561)) ([3e531b1](https://github.com/coolify-terraform/terraform-provider-coolify/commit/3e531b1bc6a069911d32bfcd3e1e3dcb96673421))

## [0.1.6](https://github.com/coolify-terraform/terraform-provider-coolify/compare/v0.1.5...v0.1.6) (2026-06-24)


### Features

* auto-update social preview image and repo description with live stats ([#531](https://github.com/coolify-terraform/terraform-provider-coolify/issues/531)) ([4fc0484](https://github.com/coolify-terraform/terraform-provider-coolify/commit/4fc0484e1172294ccfb30fb7114d5158f58a893a)), closes [#530](https://github.com/coolify-terraform/terraform-provider-coolify/issues/530)


### Bug Fixes

* address AI code quality findings ([#537](https://github.com/coolify-terraform/terraform-provider-coolify/issues/537)) ([7d8d5c1](https://github.com/coolify-terraform/terraform-provider-coolify/commit/7d8d5c1a83c8466153168e26a36fd98c3a2c9ba5))
* database health_check 422 on Coolify &lt; v4.1.2, bump min version to 4.1.0 ([#550](https://github.com/coolify-terraform/terraform-provider-coolify/issues/550)) ([01c1e57](https://github.com/coolify-terraform/terraform-provider-coolify/commit/01c1e5726dad39fba1b5d7e96628bd5317c24046))
* document RELEASE_NOTES.md must be on main, not release branch ([#527](https://github.com/coolify-terraform/terraform-provider-coolify/issues/527)) ([3e39513](https://github.com/coolify-terraform/terraform-provider-coolify/commit/3e395131bdacaa650009f0fa356fe524d1dca759)), closes [#526](https://github.com/coolify-terraform/terraform-provider-coolify/issues/526)
* release notes cleanup respects branch protection ([#529](https://github.com/coolify-terraform/terraform-provider-coolify/issues/529)) ([55b897d](https://github.com/coolify-terraform/terraform-provider-coolify/commit/55b897d97bbb564597065e67efa05334dbc33495)), closes [#526](https://github.com/coolify-terraform/terraform-provider-coolify/issues/526)
* trigger social preview update on release, add upload script ([#533](https://github.com/coolify-terraform/terraform-provider-coolify/issues/533)) ([8c2ea20](https://github.com/coolify-terraform/terraform-provider-coolify/commit/8c2ea2015df35504ebd8d601725041059c104d84)), closes [#530](https://github.com/coolify-terraform/terraform-provider-coolify/issues/530)

## [0.1.5](https://github.com/coolify-terraform/terraform-provider-coolify/compare/v0.1.4...v0.1.5) (2026-06-13)


### Features

* support RELEASE_NOTES.md override for curated release descriptions ([#525](https://github.com/coolify-terraform/terraform-provider-coolify/issues/525)) ([2597aa8](https://github.com/coolify-terraform/terraform-provider-coolify/commit/2597aa8eacaa46896a19b915670702aaf12acb8e)), closes [#524](https://github.com/coolify-terraform/terraform-provider-coolify/issues/524)
* update contract to Coolify v4.1.2 ([#518](https://github.com/coolify-terraform/terraform-provider-coolify/issues/518)) ([092ab4e](https://github.com/coolify-terraform/terraform-provider-coolify/commit/092ab4e9f5a62d04f3f7d026cb15e694b331c304)), closes [#517](https://github.com/coolify-terraform/terraform-provider-coolify/issues/517)


### Bug Fixes

* multi-perspective improvement cycle 1 ([#515](https://github.com/coolify-terraform/terraform-provider-coolify/issues/515)) ([3aa4252](https://github.com/coolify-terraform/terraform-provider-coolify/commit/3aa42527c95e0b20ead1006eb4645f33dfba8af3))
* multi-perspective improvement cycle 2 ([#520](https://github.com/coolify-terraform/terraform-provider-coolify/issues/520)) ([4bfec53](https://github.com/coolify-terraform/terraform-provider-coolify/commit/4bfec53ad983101e74bf429c65103306bfc4ca22))

## [0.1.4](https://github.com/coolify-terraform/terraform-provider-coolify/compare/v0.1.3...v0.1.4) (2026-06-03)


### Bug Fixes

* handle PATCH decode error in mock, deduplicate test mux, simplify merge target ([#508](https://github.com/coolify-terraform/terraform-provider-coolify/issues/508)) ([93fef84](https://github.com/coolify-terraform/terraform-provider-coolify/commit/93fef84f04397d3bab628e5a72f3eae79a6cdc93))
* increase polling timeout test context + bump Go 1.26.4 ([#503](https://github.com/coolify-terraform/terraform-provider-coolify/issues/503)) ([6223542](https://github.com/coolify-terraform/terraform-provider-coolify/commit/6223542987da2d73453cfb582a2c45eced18cf91))

## [0.1.3](https://github.com/coolify-terraform/terraform-provider-coolify/compare/v0.1.2...v0.1.3) (2026-06-02)


### Bug Fixes

* add coolify-v4-latest.json to .gitignore ([09bc849](https://github.com/coolify-terraform/terraform-provider-coolify/commit/09bc849d56c1c573453be4a1f223ae2b39813416))
* improve test honesty, CI safety, and error handling ([#485](https://github.com/coolify-terraform/terraform-provider-coolify/issues/485)) ([ab40882](https://github.com/coolify-terraform/terraform-provider-coolify/commit/ab408820e39d6612e7df1073b4b8c51cd1f08745))

## [0.1.2](https://github.com/coolify-terraform/terraform-provider-coolify/compare/v0.1.1...v0.1.2) (2026-06-01)


### Bug Fixes

* **ci:** exclude release-please compare URLs from lychee link check ([715124c](https://github.com/coolify-terraform/terraform-provider-coolify/commit/715124cdfac4fd1048ef93cd7232906e26449dca))

## [0.1.1](https://github.com/coolify-terraform/terraform-provider-coolify/compare/v0.1.0...v0.1.1) (2026-06-01)


### Features

* add auto-approve workflow for solo maintainer PRs ([de9e28f](https://github.com/coolify-terraform/terraform-provider-coolify/commit/de9e28ff1f3c3f734bba6b2415c9cd2dfe63a6cd)), closes [#457](https://github.com/coolify-terraform/terraform-provider-coolify/issues/457)
* adopt release-please for automated releases ([#447](https://github.com/coolify-terraform/terraform-provider-coolify/issues/447)) ([56e1add](https://github.com/coolify-terraform/terraform-provider-coolify/commit/56e1add93def5d03e9653ba222ac2f5ec310b827))


### Bug Fixes

* add deleted_at to internal fields exclusion test ([#442](https://github.com/coolify-terraform/terraform-provider-coolify/issues/442)) ([8b89d2a](https://github.com/coolify-terraform/terraform-provider-coolify/commit/8b89d2ab4bf60a77f99e41ba239bc6a6b3d3f22e))
* add make merge target and FOSSA false-positive filter ([#435](https://github.com/coolify-terraform/terraform-provider-coolify/issues/435)) ([5323f97](https://github.com/coolify-terraform/terraform-provider-coolify/commit/5323f97e6f6bba50d0ed75649856ff19ee1d925e))
* **ci:** use original filename for FOSSA CLI sha256 verification ([#440](https://github.com/coolify-terraform/terraform-provider-coolify/issues/440)) ([f2fe25f](https://github.com/coolify-terraform/terraform-provider-coolify/commit/f2fe25f096c145477a88490bcb9a8bb780304065))
* remove unused Python imports and variables ([#441](https://github.com/coolify-terraform/terraform-provider-coolify/issues/441)) ([24c928d](https://github.com/coolify-terraform/terraform-provider-coolify/commit/24c928df2db127fcbedc9b8a6b2c546852f0db6e))
* update CI job count to 9, add DCO, validate in counts-check ([#461](https://github.com/coolify-terraform/terraform-provider-coolify/issues/461)) ([d5d78c0](https://github.com/coolify-terraform/terraform-provider-coolify/commit/d5d78c015e7571d07f65aceaec240c5206bd57b2)), closes [#460](https://github.com/coolify-terraform/terraform-provider-coolify/issues/460)
* update contract with new POST /sentinel/push route ([#472](https://github.com/coolify-terraform/terraform-provider-coolify/issues/472)) ([c446a7f](https://github.com/coolify-terraform/terraform-provider-coolify/commit/c446a7fba19f593fb082b44208ff74e2ce1f4445)), closes [#471](https://github.com/coolify-terraform/terraform-provider-coolify/issues/471)
* update dependencies and pin FOSSA CLI for Scorecard ([#438](https://github.com/coolify-terraform/terraform-provider-coolify/issues/438)) ([626f0bf](https://github.com/coolify-terraform/terraform-provider-coolify/commit/626f0bf2bb343af2d725e73a703af2d84374321a))
* update stale CHANGELOG URL to current org ([e5e698a](https://github.com/coolify-terraform/terraform-provider-coolify/commit/e5e698a36e8f8f16f90234d565ec0f169adc9170))
* upgrade golang.org/x/crypto in tools module to v0.52.0 ([#466](https://github.com/coolify-terraform/terraform-provider-coolify/issues/466)) ([87c90cf](https://github.com/coolify-terraform/terraform-provider-coolify/commit/87c90cf8cf413aace9a202484dab4806388430ab))
* use stable PR author check in auto-approve workflow ([7f7599b](https://github.com/coolify-terraform/terraform-provider-coolify/commit/7f7599b5949238051ba1979fe4b2fc365166bef0))
* use workflow badge for FOSSA instead of API badge ([#437](https://github.com/coolify-terraform/terraform-provider-coolify/issues/437)) ([ca8c4c1](https://github.com/coolify-terraform/terraform-provider-coolify/commit/ca8c4c12da2863925db2fe6e3713f1394910ef76))

## [0.1.0](https://github.com/coolify-terraform/terraform-provider-coolify/releases/tag/v0.1.0) (2026-05-30)

### Breaking Changes

- `coolify_github_app`: The `private_key` attribute has been renamed to `private_key_uuid` to match the Coolify API spec. This field now accepts a UUID referencing an existing `coolify_private_key` resource instead of raw key content.
- `coolify_database_backup`: The `retain_days` attribute has been renamed to `retain_amount_locally`. The old name was misleading (it stored a count of backup copies, not days). Users must update their `.tf` files to use the new name.
- `coolify_s3_storage` resource, `coolify_s3_storage` data source, and `coolify_s3_storages` data source have been removed. Current Coolify v4 has no public top-level S3 storage API. Manage S3 storages in the Coolify web UI and reference their UUIDs from `coolify_database_backup.s3_storage_uuid`. Before upgrading, remove these from state: `terraform state rm coolify_s3_storage.<name>`.

### Added

- UUID format validation on 13 attributes across server, Hetzner, backup, scheduled task, and GitHub App resources/data sources (catches malformed input at plan time instead of API time)
- `coolify_deployment`: `wait_for_completion` attribute polls deployment status until `finished` or `error`; `timeouts` block for configurable Create timeout
- `coolify_database_backup`: 12 new fields for S3 toggle, selective backup, retention policies, and job timeout
- All application resources: 16 new fields for resource limits, health checks, and auto-deploy control
- All database and service resources: `timeouts` block with configurable Create timeout (default 10 minutes)
- 4 new singular data sources: `coolify_deployment`, `coolify_environment_variable`, `coolify_scheduled_task`, `coolify_storage`
- `tflog.Debug` structured logging in all resource CRUD methods
- **Provider configuration** with `endpoint` and `token` attributes (env var fallback: `COOLIFY_ENDPOINT`, `COOLIFY_TOKEN`)
- Health check during `Configure` validates API connection by calling `/api/v1/version`
- **Resources:**
  - `coolify_project` - Manage projects
  - `coolify_server` - Register and configure servers
  - `coolify_private_key` - Manage SSH keys
  - `coolify_application` - Deploy applications from public Git repositories
  - `coolify_application_dockerfile` - Deploy applications from Dockerfiles
  - `coolify_application_docker_image` - Deploy applications from Docker images (Docker Hub, GHCR, etc.)
  - `coolify_application_private_git` - Deploy applications from private Git repositories (SSH deploy key)
  - `coolify_application_github_app` - Deploy applications via GitHub App integration
  - `coolify_environment` - Manage project environments
  - `coolify_environment_variable` - Manage env vars for applications, services, and databases
  - `coolify_deployment` - Trigger application deployments (with `triggers` map for force-redeploy)
  - `coolify_service` - Deploy one-click services from the Coolify catalog
  - `coolify_database_postgresql` - Provision PostgreSQL databases
  - `coolify_database_mysql` - Provision MySQL databases
  - `coolify_database_mariadb` - Provision MariaDB databases
  - `coolify_database_redis` - Provision Redis databases
  - `coolify_database_mongodb` - Provision MongoDB databases
  - `coolify_database_clickhouse` - Provision ClickHouse databases
  - `coolify_database_keydb` - Provision KeyDB databases (Redis-compatible)
  - `coolify_database_dragonfly` - Provision DragonFly databases (Redis-compatible in-memory store)
  - `coolify_database_backup` - Schedule automated database backups with S3 storage and retention
  - `coolify_scheduled_task` - Manage scheduled tasks on applications/services
  - `coolify_storage` - Manage persistent storage volumes
  - `coolify_cloud_token` - Manage cloud provider tokens (Hetzner)
  - `coolify_github_app` - Manage GitHub App integrations
  - `coolify_server_hetzner` - Provision Hetzner Cloud servers via Coolify
- **Data Sources:**
  - `coolify_project` / `coolify_projects` - Read project(s)
  - `coolify_server` / `coolify_servers` - Read server(s)
  - `coolify_server_resources` - List all resources deployed on a server
  - `coolify_server_domains` - List all domains configured on a server
  - `coolify_server_validation` - Validate a server's connectivity
  - `coolify_private_key` / `coolify_private_keys` - Read SSH key(s)
  - `coolify_application` / `coolify_applications` - Read application(s)
  - `coolify_application_logs` - Read application logs
  - `coolify_database` / `coolify_databases` - Read database(s)
  - `coolify_service` / `coolify_services` - Read service(s)
  - `coolify_environment` / `coolify_environments` - Read environment(s)
  - `coolify_environment_variable` / `coolify_environment_variables` - Read / list environment variables for an application, service, or database
  - `coolify_deployment` / `coolify_deployments` - Read / list deployments for an application
  - `coolify_scheduled_task` / `coolify_scheduled_tasks` / `coolify_task_executions` - Read scheduled task(s) and executions
  - `coolify_storage` / `coolify_storages` - Read / list persistent storage volumes
  - `coolify_cloud_token` / `coolify_cloud_tokens` - Read cloud token(s)
  - `coolify_github_app` / `coolify_github_apps` / `coolify_github_app_repositories` / `coolify_github_app_branches` - Read GitHub App(s) and repos
  - `coolify_backup_executions` - List backup execution history
  - `coolify_resources` - List all resources on a server
  - `coolify_team` / `coolify_teams` / `coolify_team_members` - Read team(s) and members
  - `coolify_health` - Read Coolify instance health status
  - `coolify_version` - Read the Coolify instance version
  - `coolify_hetzner_images` / `coolify_hetzner_locations` / `coolify_hetzner_server_types` / `coolify_hetzner_ssh_keys` - Read Hetzner cloud resources
- All stateful resources support `terraform import` (action/validation resources are lifecycle-only)
- 99%+ Coolify v4 API coverage (134/135 endpoints)
- OpenAPI spec-driven test validation with libopenapi-validator
- API coverage tracking with auto-generated `API_COVERAGE.md`
- UUID format validators on all UUID input fields
- Retryable HTTP client with automatic retry on 429/5xx (3 retries, 30s timeout)
- Input validators: `build_pack` OneOf, FQDN format, cron syntax, port range (1-65535), UUID format, environment variable name format
- Configurable `timeouts` block on all application resources
- Graceful handling of out-of-band resource deletion (404 in Read removes from state)
- 750+ unit tests with race detection across 40 packages
- CI pipeline: 8 jobs (detect changes, test, lint, validate, scenario tests, acceptance tests, spec freshness, CI gate)
- GoReleaser config for GPG-signed releases
- Computed `status` field on all application resources
- Full-stack deployment example

### Changed

- `redeploy_on_update` now triggers a restart for all configuration fields including `name`, `description`, webhook secrets, auto-deploy settings, and container label options. Previously only runtime-affecting fields (ports, limits, health checks, build settings) were covered. Only immutable, computed-only, and the `redeploy_on_update` flag itself are excluded.
- `dockerfile` and `docker_compose_raw` attributes are now marked `Sensitive` (they can contain embedded secrets such as build arguments or service credentials)
- Redundant `UseStateForUnknown` plan modifier removed from `deployment_queue_limit` on server resources (the `Default` value already handles this; no user-visible behavior change)
- Consolidated `is_include_timestamps`, `enable_ssl`, and `ssl_mode` handling into shared database helpers, reducing duplication across all 8 database resources
- Minimum Terraform version requirement updated to >= 1.6 (consistent across all documentation)
- Added TRACE-level logging to version and health check endpoints for easier connection debugging
- `coolify_github_app`: `app_id`, `installation_id`, `client_id`, `client_secret`, `private_key_uuid`, and `organization_name` can now be updated in-place (previously forced destroy/recreate). This matches the Coolify API's PATCH support for these fields.
- `coolify_application_github_app`: `github_app_uuid` can now be updated in-place (previously forced destroy/recreate).

### Fixed

- API response bodies are now redacted in TRACE logs, preventing sensitive fields (passwords, keys) from appearing in debug output
- Custom TLS configuration (`ca_cert`, `insecure`) no longer silently disables HTTP retry logic
- `redactJSON` now handles JSON arrays and nested objects (previously only top-level objects were redacted)

- `coolify_service` resource: changing `name`, `description`, or `environment_name` now triggers destroy/recreate (previously produced an "Update not supported" error during apply)
- `coolify_database_clickhouse`: `clickhouse_admin_user` and `clickhouse_admin_password` are now sent during resource creation (previously silently ignored, only applied on update)
- All 8 database resources: removing `description` from config no longer leaves stale values in state (now correctly sets null when API returns empty)
- All 8 database resources: `environment_name` now has `RequiresReplace` (changing it forces a new resource, matching the API's actual behavior)
- `coolify_storage` resource: `UpdateStorageInput` now includes `UUID` field so PATCH correctly identifies the target storage
- `coolify_deployment` resource: `GetDeployment` errors during Create now produce a warning diagnostic instead of silently defaulting to "queued" status
- `coolify_private_key` resource: empty description from API now correctly becomes `null` in state (consistent with all other resources)
- `PollUntilDeleted` (used by application and service Delete) now respects the parent context's deadline instead of always using a hardcoded 2-minute timeout. Resources with a `timeouts` block now have their configured timeout honored during delete polling.
