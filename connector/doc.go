/*
The Connector is the most fundamental construct of the framework. It provides key capabilities (or building blocks)
to microservices deployed on the `Microbus`:
(1) Startup and shutdown with corresponding callbacks;
(2) Service host name and a random instance ID, both used to address the microservice;
(3) Connectivity to NATS;
(4) HTTP request/response model over NATS, both incoming (server) and outgoing (client);
(5) Logger;
(6) Configuration
*/
package connector
