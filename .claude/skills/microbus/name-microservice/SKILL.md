---
name: name-microservice
description: How to choose the hostname of a new Microbus microservice. Referenced by the add-microservice scaffolding skill (and, through their delegation to it, by add-python-microservice and import-openapi-microservice). Consult it whenever a microservice's hostname is being chosen.
---

A microservice's hostname is its unique address on the bus. It expresses where the microservice sits in the
system's domain hierarchy, not what kind of operations it performs.

## The convention

Compose the hostname from dotted segments in the form `myservice.mydomain.myapp.mycompany`, ordered most-specific
first. Only `myservice` is required; each trailing segment is optional and included only when it earns its place:

- **`myservice`** (required) - names what the microservice does. When a microservice is dedicated to a single role,
  fold the role word into this segment rather than making it a separate dotted segment: `usersui` for a UI,
  `orderscrud` for a CRUD store, `planneragent` for an agent. A separate `.ui` or `.agent` segment would read as a
  domain and is wrong - the hostname carries domain, not operation type.
- **`mydomain`** (optional) - groups microservices that belong to the same domain under a shared suffix, relating
  them to each other. Omit it for small projects; add deeper levels (`myservice.mysubdomain.mydomain`) when finer
  grouping helps.
- **`myapp`** (optional) - the application name, isolating one app from other apps that share a single NATS cluster.
  Omit it when a single app owns the cluster.
- **`mycompany`** (optional) - a trailing company suffix, used only for microservices meant to be consumed by other
  companies. Omit it for microservices internal to a single company's project.

Only letters `a-z`, digits `0-9`, hyphens `-`, and the dot `.` separator are allowed, and the hostname must be
unique across the application.

## Deriving the hostname in this repository

Derive the segments from the module path up to and including the project name. For module path
`github.com/mycompany/myproject/some/path/myservice`, the hostname is `myservice.path.some.myproject` - the
directories under the project supply the domain and app grouping, and the company segment is dropped because the
services are internal.
