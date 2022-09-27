# Encapsulaton Pattern

The `Microbus` framework aims to provide a consistent experience to developers, the users of the framework. It is opinionated about the interfaces (APIs) that are exposed to the developer and therefore opts to encapsulate underlying technologies behind its own interfaces.

One examples of this approach in this milestone is the handling of the config. Rather than leave things up to each individual developer how to fetch config values, the framework defines an interface that encapsulates the underlying implementation. A similar approach was taken with the logger.

In addition to the consistent developer experience, there are various technical reasons behind this philosophy:
* Enforcing uniformity across all microservices brings familiarity when looking at someone else's code, lowers the learning curve, and ultimately increases velocity
* The underlying technology can be changed with little impact to the microservices. For example, the source of the configs can be extended to include a remote config service in addition to the environment or the file system
* Oftentimes the underlying technology is more extensive than the basic functionality that is needed by the framework. Encapsulating the underlying API enables exposing only certain functions to the developer. For example, the logger for now is limited to only `LogInfo` and `LogError`
* The framework is in control of when and how the underlying technology is initialized. For example, future milestones will customize the logger based on the runtime environment (PROD, LAB, LOCAL, etc)
* The framework is able to seamlessly integrate building blocks together. This will take shape as more building blocks are introduced. A simple example in this milestone is how the detected config keys are logged during startup
* Bugs or CVEs in the underlying technologies are quicker to fix because there is only one source of truth. A bug such as Log4Shell (CVE-2021-44228) would require no code changes to the microservices, only to the framework
