There are many circuit breaker implementations available, one is ```spring-cloud-starter-netflix-hystrix```.
After adding it to the dependency section of your build tooling (e.g. maven) you can easily configure it via annotations.


```java
% startHighlight %
@HystrixCommand(fallbackMethod = "reliable")
% endHighlight %
public String readingList() {
	URI uri = URI.create("${target.application.http-outgoing-calls[0]:normal}....");
	return this.restTemplate.getForObject(uri, String.class);
}
% startHighlight %
public String reliable() {
	return "Steadybit";
}
% endHighlight %
```

### Downstream Endpoints
Ensure to configure circuit breakers for each of the following downstream endpoints:
${target.application.http-outgoing-calls[]:ul}


### Read More
- [Spring Cloud Circuit Breaker Guide](https://spring.io/guides/gs/cloud-circuit-breaker/)
- [CircuitBreaker by Martin Fowler](https://martinfowler.com/bliki/CircuitBreaker.html)

