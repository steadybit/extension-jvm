The circuit breaker avoid a cascading failure in case ${target.application.http-outgoing-calls[]} is unresponsive. Then, the circuit breaker triggers a fallback and allows only a small portion of requests to hit ${target.application.http-outgoing-calls[]}. Up until a normal response time is detected the fallback remains active.

[Spring Circuit Breaker Guide](https://spring.io/guides/gs/circuit-breaker/)
