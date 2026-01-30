# Changelog

## 0.8.0 (2026-01-30)

Full Changelog: [v0.7.0...v0.8.0](https://github.com/kernel/hypeman-cli/compare/v0.7.0...v0.8.0)

### ⚠ BREAKING CHANGES

* add support for passing files as parameters

### Features

* add better suggests when commands don't match ([dc7d5d6](https://github.com/kernel/hypeman-cli/commit/dc7d5d69db1bb85b69d9c7a5d2f27df8eb7708d8))
* add hypeman build command ([#21](https://github.com/kernel/hypeman-cli/issues/21)) ([27a15cf](https://github.com/kernel/hypeman-cli/commit/27a15cf3bbe004f842c9cf2a2e241b422105dc2c))
* add hypeman cp for file copy to/from running VMs ([a198210](https://github.com/kernel/hypeman-cli/commit/a198210b55255d43734b32fa1c949004d0dcdd6a))
* add support for passing files as parameters ([c0fbb51](https://github.com/kernel/hypeman-cli/commit/c0fbb51d43492142b55d2c81d21298927738bd35))
* added mock server tests ([311ee96](https://github.com/kernel/hypeman-cli/commit/311ee96f01018480354a628449e32e51c5b16016))
* added support for --foo.baz inner field flags ([01a996a](https://github.com/kernel/hypeman-cli/commit/01a996a1af94be45f9a87dbfb6565bcaf252318e))
* **api:** manual updates ([7c2e7c0](https://github.com/kernel/hypeman-cli/commit/7c2e7c0ac2c82677e41c1a0194bc70b320f129b9))
* **api:** manual updates ([7159a9e](https://github.com/kernel/hypeman-cli/commit/7159a9e0a86185b596b097ef46cdaa368d46cf05))
* **cli:** binary request bodies ([25d5a4f](https://github.com/kernel/hypeman-cli/commit/25d5a4f1390dcf6c62546ee6620e916008f5eafb))
* **cli:** improve shell completions for namespaced commands and flags ([e27a3a6](https://github.com/kernel/hypeman-cli/commit/e27a3a65a0b35e869b62341927fa0e79790af401))
* enable CI tests ([62f708e](https://github.com/kernel/hypeman-cli/commit/62f708ef02861f1852c1e0843bd78d3b568304ef))
* enable suggestion for mistyped commands and flags ([7bfa7fe](https://github.com/kernel/hypeman-cli/commit/7bfa7fe31083c1b8af021b3aaeefd071ca7dad10))
* enable support for streaming endpoints ([859ee0b](https://github.com/kernel/hypeman-cli/commit/859ee0b063183e45fcbaa60328dc77670ccb203c))
* gpu passthrough ([1792fbd](https://github.com/kernel/hypeman-cli/commit/1792fbdc848fa332eecd0a19f86171b89f4e3b11))
* improved behavior for exploring paginated/streamed endpoints ([24700e8](https://github.com/kernel/hypeman-cli/commit/24700e8a7fc00c93eabe2c96e1da764c0aca6266))
* new and simplified CLI flag parsing code and YAML support ([b7b4631](https://github.com/kernel/hypeman-cli/commit/b7b4631a26b0ab2059eae9e7948a6411b03d3196))
* Operational logs over API: hypeman.log, vmm.log ([78d5b3d](https://github.com/kernel/hypeman-cli/commit/78d5b3d89df2ae69c8a17d560916226262c2d974))
* QEMU support ([92c4e94](https://github.com/kernel/hypeman-cli/commit/92c4e94394a817ee9a072f2a2fe5dc75075e701d))
* redact `Authorization` header when using debug option ([29b13d1](https://github.com/kernel/hypeman-cli/commit/29b13d104a64890acd5aa11aabba2006c13948d9))
* redact secrets from other authentication headers when using debug option ([b1005cc](https://github.com/kernel/hypeman-cli/commit/b1005cc21496d6fbae11da5885e815424335784b))
* Resource accounting ([1a4f78f](https://github.com/kernel/hypeman-cli/commit/1a4f78f6a6b72a0e5be358b0d9d222cc147cfc5f))
* Start and Stop VM ([5955215](https://github.com/kernel/hypeman-cli/commit/5955215e16cedf91f7a925437d434f5912de26da))
* Start and Stop VM ([f3953fb](https://github.com/kernel/hypeman-cli/commit/f3953fbfde05295f76c96ddde76d2408d1812371))
* Start and Stop VM ([9681f6d](https://github.com/kernel/hypeman-cli/commit/9681f6d3f25f5646aa46d3f3fc71efb7edfdc2de))


### Bug Fixes

* address multi-platform build ([e8473b2](https://github.com/kernel/hypeman-cli/commit/e8473b2b76a9939e42b9ff3f8160cafe80fe03d4))
* check required arguments ([1027f9c](https://github.com/kernel/hypeman-cli/commit/1027f9c8b50e60af3501fb877210e82c6f7de064))
* **client:** do not use pager for short paginated responses ([739a322](https://github.com/kernel/hypeman-cli/commit/739a322a2135fbc41a6ccb7c33bfee99d8ad43e8))
* **cli:** fix compilation on Windows ([52bc062](https://github.com/kernel/hypeman-cli/commit/52bc0621c9e3ff4711160d7566b8a87f3b66537d))
* **cli:** remove `*.exe` files from customer SDK changes ([c82e721](https://github.com/kernel/hypeman-cli/commit/c82e721b331693e78aef2a86458106a828a54458))
* fix for empty request bodies ([f405d81](https://github.com/kernel/hypeman-cli/commit/f405d810e062dcaaf9c7ce829f6a135e43ed1a7c))
* fix for paginated output not writing to pager correctly ([cdeb9a8](https://github.com/kernel/hypeman-cli/commit/cdeb9a87e922c9011dd197210992a6e96e3e50c1))
* fix generated flag types and value wrapping ([5094864](https://github.com/kernel/hypeman-cli/commit/50948647277722f818f7c5aec3ff9dfd29d0d923))
* fix mock tests with inner fields that have underscores ([6a86f0d](https://github.com/kernel/hypeman-cli/commit/6a86f0dd2b6799a8bfed71464b395aea738d805a))
* fixed manpage generation ([0068246](https://github.com/kernel/hypeman-cli/commit/0068246311dc8aed0c1c61e1a474cc66b3823f9c))
* fixed placeholders for date/time arguments ([5043343](https://github.com/kernel/hypeman-cli/commit/504334379170e58205e768c0673abe9340716d7e))
* **go.mod:** remove replace ([1cb7f29](https://github.com/kernel/hypeman-cli/commit/1cb7f29107ba9d77792974f5ce52548e9b41af4c))
* **go.mod:** replace onkernel with kernel ([0c4d4df](https://github.com/kernel/hypeman-cli/commit/0c4d4df572f5cdb60c2c8187bf32f4286db0ae97))
* ignore .exe files ([96792b4](https://github.com/kernel/hypeman-cli/commit/96792b4f909c5d7354b02b4f2666528ff917c676))
* image name encoding and auth on push ([#28](https://github.com/kernel/hypeman-cli/issues/28)) ([d177c91](https://github.com/kernel/hypeman-cli/commit/d177c9198fabbf34fe4cf7d6680b4c1d451217b8))
* **mcp:** correct code tool API endpoint ([dadfae0](https://github.com/kernel/hypeman-cli/commit/dadfae07657bd51ed4eb8e1e5093e0342e483434))
* paginated endpoints now behave better with pagers by default ([85bcddf](https://github.com/kernel/hypeman-cli/commit/85bcddf4f61de98a376f52d47dbf76a24e1f9114))
* remove unsupported methods ([2d143b6](https://github.com/kernel/hypeman-cli/commit/2d143b60b512d39294ef3257423fbc485eda2f78))
* restore support for void endpoints ([902cbe3](https://github.com/kernel/hypeman-cli/commit/902cbe3c8c10265fd2edb04ff2bdaf48b2eca4aa))
* update SDK to v0.9.1 and fix goreleaser config ([#25](https://github.com/kernel/hypeman-cli/issues/25)) ([8022e05](https://github.com/kernel/hypeman-cli/commit/8022e05e19046c2e7a049db4fb1cba361ec36046))


### Chores

* **cli:** add `*.exe` files back to `.gitignore` ([bf6fd1c](https://github.com/kernel/hypeman-cli/commit/bf6fd1c975c8ae31b59a1ea7bff2d73d3e95d41d))
* **cli:** move `jsonview` subpackage to `internal` ([94acefd](https://github.com/kernel/hypeman-cli/commit/94acefdf272d96b5ca9d8d28f13426b6497440aa))
* **cli:** run pre-codegen tests on Windows ([3393647](https://github.com/kernel/hypeman-cli/commit/339364712f3d35d098efe7c2e2431522608bd380))
* **cli:** temporarily remove `*.exe` from `.gitignore` ([2f1e331](https://github.com/kernel/hypeman-cli/commit/2f1e331700a8e564f593c23ecfa2e30ef4736482))
* **internal:** codegen related update ([c3cef92](https://github.com/kernel/hypeman-cli/commit/c3cef92f505cec2e14547c8505a3ae1f2ecaabfc))
* **internal:** codegen related update ([efd7014](https://github.com/kernel/hypeman-cli/commit/efd701434f04d97fec613b9713f53894d2605afd))
* **internal:** codegen related update ([3039412](https://github.com/kernel/hypeman-cli/commit/3039412830a7d98b4667591742f9127cf9dc1114))
* **internal:** codegen related update ([4560236](https://github.com/kernel/hypeman-cli/commit/4560236ceccefbfc2b393009226f1f1117c98f4e))
* **internal:** codegen related update ([aee5c33](https://github.com/kernel/hypeman-cli/commit/aee5c339da85dee0907f9726d475ace42261f835))
* **internal:** codegen related update ([8488fd9](https://github.com/kernel/hypeman-cli/commit/8488fd91e261aa94be6cb992d7b532dcdec28ac6))
* **internal:** codegen related update ([ab11523](https://github.com/kernel/hypeman-cli/commit/ab11523b1d74a63a7fe933e75f11f8384fe2ed66))
* **internal:** codegen related update ([e7a637d](https://github.com/kernel/hypeman-cli/commit/e7a637da57cddc391631a91fdc4b9fe54ac57c80))
* **internal:** codegen related update ([c6753bd](https://github.com/kernel/hypeman-cli/commit/c6753bdc19dc0ec7582ee47fb2ad8877f7f2b294))
* **internal:** codegen related update ([7264dd4](https://github.com/kernel/hypeman-cli/commit/7264dd408c234b058a315704a86681bd2edd6e57))
* **internal:** codegen related update ([33988da](https://github.com/kernel/hypeman-cli/commit/33988da5ed8de52693b11235e8b3fce7a448f024))
* **internal:** codegen related update ([8002ef7](https://github.com/kernel/hypeman-cli/commit/8002ef7e6a60cc5f0c3c087d752e41c579602771))
* **internal:** codegen related update ([91c9d86](https://github.com/kernel/hypeman-cli/commit/91c9d860a3718140270a4aae7a91d35b5229e013))
* **internal:** codegen related update ([60d1219](https://github.com/kernel/hypeman-cli/commit/60d1219d51665f9f2f3fc0d24cb5f20195a110d0))
* **internal:** codegen related update ([b2d2061](https://github.com/kernel/hypeman-cli/commit/b2d2061f8163d3f5331b65057f3321b017e2642c))
* **internal:** version bump ([96d512f](https://github.com/kernel/hypeman-cli/commit/96d512ffb6c21a74478d3bb0c9646fea39a58517))
* rename GitHub org from onkernel to kernel ([#23](https://github.com/kernel/hypeman-cli/issues/23)) ([d107d16](https://github.com/kernel/hypeman-cli/commit/d107d16afa0d1c5a5acc2737cf7f1a761c90836b))
* rename org from onkernel to kernel ([0b5fe7c](https://github.com/kernel/hypeman-cli/commit/0b5fe7c5beefc1191fe255e082569a735129d61c))
* update Go SDK version ([23fc553](https://github.com/kernel/hypeman-cli/commit/23fc5534efe1acbbd0a3fd08602a9db8b53aace6))
* use `stretchr/testify` assertion helpers in tests ([6206fbe](https://github.com/kernel/hypeman-cli/commit/6206fbefa0b55a0be7102be506fbe0bcf647b927))

## 0.7.0 (2025-12-23)

Full Changelog: [v0.6.1...v0.7.0](https://github.com/onkernel/hypeman-cli/compare/v0.6.1...v0.7.0)

### Features

* add cp command for file copy to/from instances ([#18](https://github.com/onkernel/hypeman-cli/issues/18)) ([f67ad7b](https://github.com/onkernel/hypeman-cli/commit/f67ad7bcb6fbbe0a9409574fababab862da87840))


### Chores

* **internal:** codegen related update ([a6c6588](https://github.com/onkernel/hypeman-cli/commit/a6c6588d42a6981b65f5144d033f040afc29a959))

## 0.6.1 (2025-12-11)

Full Changelog: [v0.6.0...v0.6.1](https://github.com/onkernel/hypeman-cli/compare/v0.6.0...v0.6.1)

## 0.6.0 (2025-12-06)

Full Changelog: [v0.5.1...v0.6.0](https://github.com/onkernel/hypeman-cli/compare/v0.5.1...v0.6.0)

### Features

* **cli:** automatic streaming for paginated endpoints ([9af6924](https://github.com/onkernel/hypeman-cli/commit/9af69246d62010c32d39583c8b1eba39a663d3fa))

## 0.5.1 (2025-12-05)

Full Changelog: [v0.5.0...v0.5.1](https://github.com/onkernel/hypeman-cli/compare/v0.5.0...v0.5.1)

### Features

* **api:** manual updates ([a3f2ec1](https://github.com/onkernel/hypeman-cli/commit/a3f2ec15101a6afd6feb1da1addcb3a2589acb53))
* fix edge cases for sending request data and add YAML support ([3e740a9](https://github.com/onkernel/hypeman-cli/commit/3e740a94698f4704e79cc5c3b6434cbb1bfcb935))
* Ingress ([bfb79c5](https://github.com/onkernel/hypeman-cli/commit/bfb79c5a160a3b92cac3793ea49da49ddcc7c8c6))
* Initialize volume with data ([ef9997c](https://github.com/onkernel/hypeman-cli/commit/ef9997cc2c6d0fc14531bdf9d1238f3447e3a454))
* **push:** add hypeman push command for local image upload ([e120ec6](https://github.com/onkernel/hypeman-cli/commit/e120ec6d96531ab49909a3d55895f5fcc4d43dc2))
* respect HYPEMAN_BASE_URL environment variable ([17122d7](https://github.com/onkernel/hypeman-cli/commit/17122d7b2d6041c57d4e2d341b52f18697aef5d4))


### Bug Fixes

* fix for default flag values ([812e009](https://github.com/onkernel/hypeman-cli/commit/812e0091f73ab5e8992adab5ca1c2cef76b60c63))
* **run:** wait for image to be ready before creating instance ([048ee73](https://github.com/onkernel/hypeman-cli/commit/048ee7311c39d6c3c7efad9c662fa2a1993ced97))
* use correct user agent value ([580e468](https://github.com/onkernel/hypeman-cli/commit/580e468e95a11c8c57016954464039af3b0586f1))


### Chores

* add scripts ([c3e4955](https://github.com/onkernel/hypeman-cli/commit/c3e4955f932edc7567d929f22f3e93f22ae69e1a))
* update dependencies ([4ed31f6](https://github.com/onkernel/hypeman-cli/commit/4ed31f6294c1b94ef764bb7959dc99e89af62cfb))

## 0.5.0 (2025-11-26)

Full Changelog: [v0.4.0...v0.5.0](https://github.com/onkernel/hypeman-cli/compare/v0.4.0...v0.5.0)

### Features

* Generate log streaming ([31951c5](https://github.com/onkernel/hypeman-cli/commit/31951c5caf65c008f9811ffd023f54a10c3f1474))

## 0.4.0 (2025-11-26)

Full Changelog: [v0.3.0...v0.4.0](https://github.com/onkernel/hypeman-cli/compare/v0.3.0...v0.4.0)

### Features

* Remove exec from openapi spec ([6bde031](https://github.com/onkernel/hypeman-cli/commit/6bde031264de6cd6b17afe32f73a70bf14c2f36d))

## 0.3.0 (2025-11-26)

Full Changelog: [v0.2.0...v0.3.0](https://github.com/onkernel/hypeman-cli/compare/v0.2.0...v0.3.0)

### Features

* **api:** add exec ([cc1d174](https://github.com/onkernel/hypeman-cli/commit/cc1d17479467b19436346b30256f92d99474d9ed))

## 0.2.0 (2025-11-26)

Full Changelog: [v0.1.2...v0.2.0](https://github.com/onkernel/hypeman-cli/compare/v0.1.2...v0.2.0)

### Features

* Network manager ([cece9ba](https://github.com/onkernel/hypeman-cli/commit/cece9ba6e801a9b29512357060e5642976c8e3ec))


### Chores

* **client:** change name of underlying properties for models and params ([27fd97a](https://github.com/onkernel/hypeman-cli/commit/27fd97aa3faa8b436a625783232d36250bbd191a))

## 0.1.2 (2025-11-20)

Full Changelog: [v0.1.1...v0.1.2](https://github.com/onkernel/hypeman-cli/compare/v0.1.1...v0.1.2)

### ⚠ BREAKING CHANGES

* new logic for parsing arguments

### Features

* new logic for parsing arguments ([de05b62](https://github.com/onkernel/hypeman-cli/commit/de05b6274cb3d3c27dcfe9784a331a9762a8dca5))

## 0.1.1 (2025-11-14)

Full Changelog: [v0.1.0...v0.1.1](https://github.com/onkernel/hypeman-cli/compare/v0.1.0...v0.1.1)

### Features

* **api:** manual updates ([1133f94](https://github.com/onkernel/hypeman-cli/commit/1133f94ebcc7e53162d26aa03a265e3520806ebb))

## 0.1.0 (2025-11-14)

Full Changelog: [v0.0.1...v0.1.0](https://github.com/onkernel/hypeman-cli/compare/v0.0.1...v0.1.0)

### Features

* **api:** add homebrew ([489dbc8](https://github.com/onkernel/hypeman-cli/commit/489dbc83126ed1f9c506ec64d7f5291f3adfc0ac))
* **api:** make public ([a42708e](https://github.com/onkernel/hypeman-cli/commit/a42708e4e7f906d338b2db8da0ef56355e2b6ba8))
* **api:** manual updates ([2aa453f](https://github.com/onkernel/hypeman-cli/commit/2aa453f3c0cc4191329ade9a8c8b2b328eca97e1))
