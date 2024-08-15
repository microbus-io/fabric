# Locality-Aware Routing

A geographic locality, when provided, is used by `Microbus` to optimize routing of [unicast](../blocks/unicast.md) communications. A microservice making a unicast request will prioritize microservices that most resemble its own locality. For example, if the upstream microservice is located in `1.b.west.us` and replicas of the downstream microservice are located in both `2.b.west.us` and `1.a.east.us`, the request will be directed to the former because they share the longer common suffix `b.west.us`.

Locality-aware routing works when both upstream and downstream microservices designate a locality, by means of the `MICROBUS_LOCALITY` environment variable or the `SetLocality` method of the `Connector`. The locality is a hierarchical pattern similar to that of a standard hostname, with the most specific location first.

The special value `AWS` can be used when deployed on AWS to determine the availability zone automatically by making a request to the meta-data service at `http://169.254.169.254/latest/meta-data/placement/availability-zone`. The availability zone will then be used as the basis for the locality of the microservice. For example, availability zone `us-east-1b` is transformed to locality `b.1.east.us`.

Similarly, the special value `GCP` can be used when deployed on GCP to determine the availability zone from `http://metadata.google.internal/computeMetadata/v1/instance/zone`. For example, availability zone `us-east1-b` is transformed to locality `b.1.east.us`

Caution: When designating either `AWS` or `GCP`, the microservice will fail to start if the availability zone cannot be determined.

Using the availability zone as the basis of the locality, for example `1.b.west.us`, keeps traffic in the same availability zone, in the same data center, or in the same region, whichever is nearest. This is often the best strategy but it ultimately depends on the geographic distribution of the microservices.

<img src="locality-aware-routing-1.drawio.svg">
<p></p>

Using the machine identifier in the locality, for example `a6506f32.1.b.west.us`or even just `a6506f32`, is a more aggressive strategy that keeps traffic within the same machine when possible. This strategy reduces service-to-service latency but can lead to an imbalanced load distribution.
