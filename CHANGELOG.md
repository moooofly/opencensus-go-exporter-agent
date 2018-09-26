<a name="1.2.0"></a>
# [1.2.0](https://github.com/moooofly/opencensus-go-exporter-hunter/compare/v1.1.0...v1.2.0) (2018-09-26)


### Features

* change exporter.Flush() to exporter.Stop() ([45bd923](https://github.com/moooofly/opencensus-go-exporter-hunter/commit/45bd923))
* support package management by glide ([287c655](https://github.com/moooofly/opencensus-go-exporter-hunter/commit/287c655))
* update Dockerfile to multi-stage since vendor support ([23a5385](https://github.com/moooofly/opencensus-go-exporter-hunter/commit/23a5385))



<a name="1.1.0"></a>
# [1.1.0](https://github.com/moooofly/opencensus-go-exporter-hunter/compare/v1.0.0...v1.1.0) (2018-09-25)


### Features

* update codes by using ocgrpc-wrapper ([0f5d5a8](https://github.com/moooofly/opencensus-go-exporter-hunter/commit/0f5d5a8))



<a name="1.0.0"></a>
# [1.0.0](https://github.com/moooofly/opencensus-go-exporter-hunter/compare/v0.11.0...v1.0.0) (2018-09-20)


### Features

* rename opencensus-go-exporter-agent to opencensus-go-exporter-hunter ([01304aa](https://github.com/moooofly/opencensus-go-exporter-hunter/commit/01304aa))



<a name="0.11.0"></a>
# [0.11.0](https://github.com/moooofly/opencensus-go-exporter-hunter/compare/v0.10.0...v0.11.0) (2018-09-20)


### Features

* modularization support and initial connect retry support ([66ec2e9](https://github.com/moooofly/opencensus-go-exporter-hunter/commit/66ec2e9))



<a name="0.10.0"></a>
# [0.10.0](https://github.com/moooofly/opencensus-go-exporter-hunter/compare/v0.9.0...v0.10.0) (2018-09-18)


### Bug Fixes

* fix missing error-rate setting ([6fcdb51](https://github.com/moooofly/opencensus-go-exporter-hunter/commit/6fcdb51))


### Features

* support batch send by bundle package ([68a1211](https://github.com/moooofly/opencensus-go-exporter-hunter/commit/68a1211))


### Performance Improvements

* add '>/dev/null 2>&1' to cmd in Makefile ([d9a6f39](https://github.com/moooofly/opencensus-go-exporter-hunter/commit/d9a6f39))
* remove `time.Sleep()` from local_example/main.go ([84c16b8](https://github.com/moooofly/opencensus-go-exporter-hunter/commit/84c16b8))



<a name="0.9.0"></a>
# [0.9.0](https://github.com/moooofly/opencensus-go-exporter-hunter/compare/v0.8.0...v0.9.0) (2018-09-06)


### Bug Fixes

* fix wrong .gitignore setting ([2f5990a](https://github.com/moooofly/opencensus-go-exporter-hunter/commit/2f5990a))


### Features

* add error_rate support ([b31c46d](https://github.com/moooofly/opencensus-go-exporter-hunter/commit/b31c46d))



<a name="0.8.0"></a>
# [0.8.0](https://github.com/moooofly/opencensus-go-exporter-hunter/compare/v0.7.0...v0.8.0) (2018-08-30)


### Features

* **example:** add callchain example ([96b2b24](https://github.com/moooofly/opencensus-go-exporter-hunter/commit/96b2b24))



<a name="0.7.0"></a>
# [0.7.0](https://github.com/moooofly/opencensus-go-exporter-hunter/compare/v0.6.0...v0.7.0) (2018-08-28)


### Features

* **example:** remove MethodName setting, update ServiceName setting, remove sleep span ([cc1d689](https://github.com/moooofly/opencensus-go-exporter-hunter/commit/cc1d689))
* **example:** use NewClientCustomInfo/NewServerCustomInfo for easy attributes setting ([41dd0f0](https://github.com/moooofly/opencensus-go-exporter-hunter/commit/41dd0f0))



<a name="0.6.0"></a>
# [0.6.0](https://github.com/moooofly/opencensus-go-exporter-hunter/compare/v0.5.0...v0.6.0) (2018-08-27)


### Features

* **example:** add Dockerfile for grpc_example ([86958af](https://github.com/moooofly/opencensus-go-exporter-hunter/commit/86958af))
* **example:** update grpc_example/local_example to obtain config from env and file ([6dd2565](https://github.com/moooofly/opencensus-go-exporter-hunter/commit/6dd2565))
* **example:** update src files of grpc_example for dockerization ([56626dd](https://github.com/moooofly/opencensus-go-exporter-hunter/commit/56626dd))
* update Makefile for easy docker test ([6a299cd](https://github.com/moooofly/opencensus-go-exporter-hunter/commit/6a299cd))



<a name="0.5.0"></a>
# [0.5.0](https://github.com/moooofly/opencensus-go-exporter-hunter/compare/v0.4.0...v0.5.0) (2018-08-24)


### Features

* **example:** add real grpc service call-chain example ([edb2969](https://github.com/moooofly/opencensus-go-exporter-hunter/commit/edb2969))
* make agent fit for both trace exporter and view exporter ([87bae0b](https://github.com/moooofly/opencensus-go-exporter-hunter/commit/87bae0b))



<a name="0.4.0"></a>
# [0.4.0](https://github.com/moooofly/opencensus-go-exporter-hunter/compare/v0.3.0...v0.4.0) (2018-08-20)


### Features

* add Attributes/TimeEvents/Status metadata support ([b0d6746](https://github.com/moooofly/opencensus-go-exporter-hunter/commit/b0d6746))
* support unix socket, update Makefile, optimize codes ([75c23bf](https://github.com/moooofly/opencensus-go-exporter-hunter/commit/75c23bf))



<a name="0.3.0"></a>
# [0.3.0](https://github.com/moooofly/opencensus-go-exporter-hunter/compare/v0.2.0...v0.3.0) (2018-08-15)


### Features

* remove dump.proto, add exporter.proto, change codes accordingly ([97e3981](https://github.com/moooofly/opencensus-go-exporter-hunter/commit/97e3981))



<a name="0.2.0"></a>
# [0.2.0](https://github.com/moooofly/opencensus-go-exporter-hunter/compare/v0.1.0...v0.2.0) (2018-08-14)


### Features

* optimize tcp and unix endpoints chosen strategy ([05cd2e2](https://github.com/moooofly/opencensus-go-exporter-hunter/commit/05cd2e2))


<a name="0.1.0"></a>
# [0.1.0](https://github.com/moooofly/opencensus-go-exporter-hunter/compare/3f50a2b...v0.1.0) (2018-08-13)


### Features

* **whole project:** create a basis of opencensus exporter for Hunter Agent ([b758e70](https://github.com/moooofly/opencensus-go-exporter-hunter/commit/b758e70))


