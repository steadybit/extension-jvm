A circuit breaker avoids a cascading failure if a downstream endpoint is unresponsive. By having a fallback for the actual downstream call, the circuit breaker only allows a small portion of requests to go through in case the downstream requests fail. Until a certain amount of successful downstream responses, the fallback remains active.

***More Resources***
* [CircuitBreaker by Martin Fowler](https://martinfowler.com/bliki/CircuitBreaker.html)
