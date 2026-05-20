# Changelog

## [1.4.0](https://github.com/aaronflorey/genignore/compare/v1.3.0...v1.4.0) (2026-05-20)


### Features

* **api:** fetch gitignore templates from github ([e87f0a6](https://github.com/aaronflorey/genignore/commit/e87f0a6d2015b09baad349c905d2a8da2122cae2))
* **app:** fan out detect across packages subfolders ([d76222f](https://github.com/aaronflorey/genignore/commit/d76222f4af6b01e27d69cb4549465c31659363d2))
* **config:** Add machine-level defaults and Wrangler support ([74e9242](https://github.com/aaronflorey/genignore/commit/74e92427ec0ff50f7beceab307556b103309a440))
* **custom-template:** Add embedded custom gitignore templates ([6b9889a](https://github.com/aaronflorey/genignore/commit/6b9889a7fe682a6a83cb8e5f51505ad73eb8cab3))
* initial commit ([42fab5b](https://github.com/aaronflorey/genignore/commit/42fab5b3fddb9a0364de540daee20822fce77600))
* **provider:** Detect one-level project signals ([3bf9147](https://github.com/aaronflorey/genignore/commit/3bf91470d2808be2c94bf59f17e75a6648e0b759))
* **provider:** Skip ignored directories during detection ([ed92394](https://github.com/aaronflorey/genignore/commit/ed92394940d1f45993e01a0105193809821f54d0))


### Fixes

* **08:** close audit and detect scope gaps ([fac9ee4](https://github.com/aaronflorey/genignore/commit/fac9ee4fbe248a5f5e7b4b471c33514926342bfc))
* **09:** allow embedded-only degraded catalog flow ([3aea8ab](https://github.com/aaronflorey/genignore/commit/3aea8abf5a6bb2208aa43612760e2d38d11ad76b))
* **10:** reuse provider catalog snapshot ([63bf236](https://github.com/aaronflorey/genignore/commit/63bf236a3bfebac93985bf1eef6fc8bd06c37b54))
* **11:** make managed env ownership explicit ([9af7a40](https://github.com/aaronflorey/genignore/commit/9af7a40a0d52befc0927c27a10080369734786cd))
* **12:** report no-op gitignore reruns ([4b3fcd5](https://github.com/aaronflorey/genignore/commit/4b3fcd543adb44057bef9caac31b67477130c886))
* **13:** validate pinned release packaging ([b33b553](https://github.com/aaronflorey/genignore/commit/b33b5535e0d4c7fa94babf623b7f3579f419a63c))
* **14:** close remaining milestone audit blockers ([e4be78c](https://github.com/aaronflorey/genignore/commit/e4be78c83daaeb772244c457e752cdc3bc77b258))
* **14:** remove implied env config wiring ([6032e89](https://github.com/aaronflorey/genignore/commit/6032e89acb89e2aff582f13c3c638493bcb1184a))
* **15:** add binary coverage and stable detection ([18c6fc2](https://github.com/aaronflorey/genignore/commit/18c6fc2b15cf3dd0774ddd5c6eab5b87d80c4dee))
* **15:** add gitignore fuzz and benchmarks ([30ea2a2](https://github.com/aaronflorey/genignore/commit/30ea2a2bd7b2563154c201fcf1a8fe7b30dccdbf))
* **15:** add offline runtime cache support ([6aac859](https://github.com/aaronflorey/genignore/commit/6aac859174020910a4f492b702e1d2a9138333fb))
* **15:** add release artifact smoke checks ([d9d5e31](https://github.com/aaronflorey/genignore/commit/d9d5e31dbbff8f2826d3a68a6cc72df8ca32ce0a))
* **16:** add doctor visibility and diff preview ([a0dec19](https://github.com/aaronflorey/genignore/commit/a0dec1969d8a0ffb0de6618fe440cde175973e4c))
* **16:** add fixture corpus and output contracts ([0a274f9](https://github.com/aaronflorey/genignore/commit/0a274f9afdcbd394f3510b796a600d4860c6146a))
* **16:** add read-only provider resolution ([a188bc5](https://github.com/aaronflorey/genignore/commit/a188bc52fe5f726a7986bddb9a76be9400cf18e5))
* **16:** hide detect diff after write ([81ecc39](https://github.com/aaronflorey/genignore/commit/81ecc39112f1d28a22accd1789812ca59e42dc10))
* **16:** pin upstream cache validation ([5ba22de](https://github.com/aaronflorey/genignore/commit/5ba22dedcc738a90de7a2760e370cc032a561efa))
* **16:** refresh provider catalog snapshot ([0418021](https://github.com/aaronflorey/genignore/commit/04180215050d84be1ef253b4e463a3a4acd9a874))
* **runtime:** source provider catalog at runtime ([8b5a498](https://github.com/aaronflorey/genignore/commit/8b5a4982ab505204c4970b563e9387b33fd2179f))

## [1.3.0](https://github.com/aaronflorey/genignore/compare/v1.2.0...v1.3.0) (2026-04-13)


### Features

* **app:** fan out detect across packages subfolders ([fdc443b](https://github.com/aaronflorey/genignore/commit/fdc443bea4e7b2f019c219976e9c78ea43d22fc1))
* initial commit ([e762f19](https://github.com/aaronflorey/genignore/commit/e762f192863c2575b62a8e29712de1f554b6a4ef))
* **provider:** Detect one-level project signals ([63796db](https://github.com/aaronflorey/genignore/commit/63796dbfa9208730921592fca905bad38de7f425))

## [1.2.0](https://github.com/aaronflorey/genignore/compare/v1.1.0...v1.2.0) (2026-04-13)


### Features

* initial commit ([e762f19](https://github.com/aaronflorey/genignore/commit/e762f192863c2575b62a8e29712de1f554b6a4ef))
* **provider:** Detect one-level project signals ([63796db](https://github.com/aaronflorey/genignore/commit/63796dbfa9208730921592fca905bad38de7f425))

## [1.1.0](https://github.com/aaronflorey/genignore/compare/v1.0.1...v1.1.0) (2026-04-12)


### Features

* **260411-dnn:** add cross-platform IDE and JetBrains inference detection ([ded7ac1](https://github.com/aaronflorey/genignore/commit/ded7ac13cfb4af128655cd2270df8a705a8b0d01))
* **260411-dww:** expand JetBrains language-aware IDE detection ([09bed7b](https://github.com/aaronflorey/genignore/commit/09bed7b71be71885f67702f841d89bbc6ca40fa2))
* **provider:** Add detectors for mainstream templates ([9b2499a](https://github.com/aaronflorey/genignore/commit/9b2499a7c77d1ad55de5cd964b4864bc4deed923))

## [1.0.1](https://github.com/aaronflorey/genignore/compare/v1.0.0...v1.0.1) (2026-04-11)


### Fixes

* **260411-anz:** preserve managed OS providers during detect ([9a8770b](https://github.com/aaronflorey/genignore/commit/9a8770b46ebf8819522a3a41898fc0a3e201507a))

## [1.0.0](https://github.com/aaronflorey/genignore/compare/v1.0.0...v1.0.0) (2026-04-11)


### meta

* **repo:** Align release and planning configuration ([f2ac69b](https://github.com/aaronflorey/genignore/commit/f2ac69bfa716f0161365222fcc6123cffafbae4c))


### Features

* change name of binary ([f568e9e](https://github.com/aaronflorey/genignore/commit/f568e9e5c55a46eed2e56771768de97484699e83))


### Fixes

* **260411-47x:** dedupe unmanaged rules against managed block ([bb00825](https://github.com/aaronflorey/genignore/commit/bb008251a2cb95a234f3a85aacf7bf62bdeb2861))
* **quick-260411-989:** document CI checks and unblock lint ([8cd813b](https://github.com/aaronflorey/genignore/commit/8cd813b6723dd816bc45fe32c444a1d687b51fa5))
