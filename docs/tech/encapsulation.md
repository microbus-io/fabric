# Encapsulation Pattern

`Microbus` employs the principle of information hiding and opts to encapsulate the underlying technologies behind its own simplified interfaces. There are various reasons for this pattern:

* Providing a cohesive experience to developers
* Enforcing uniformity across all microservices brings familiarity when looking at someone else's code, lowers the learning curve, and ultimately increases velocity
* The underlying technology can be changed with little impact to the microservices
* Oftentimes the underlying technology is more extensive than the basic functionality that is needed by the framework. Encapsulating the underlying API enables exposing only certain functions to the developer
* The framework is in control of when and how the underlying technology is initialized
* The framework is able to seamlessly integrate building blocks together. This will take shape as more building blocks are introduced
* Bugs or CVEs in the underlying technologies are quicker to fix because there is only one source of truth. A bug such as [Log4Shell CVE-2021-44228](https://logging.apache.org/log4j/2.x/security.html) would require no code changes to the microservices, only to the framework

One example of this pattern is with the configuration of microservices. Rather than leave things up to each individual developer how to fetch config values, the `Connector` defines an interface that encapsulates the underlying implementation. Today, the framework looks for config values in a `config.yaml` file. In the future, it might be extended to fetch configs from a remote location.
