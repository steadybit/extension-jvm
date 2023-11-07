There are many circuit breaker implementations available, one is ```spring-cloud-starter-netflix-hystrix```. After adding it to the dependency section of your build tooling (e.g. maven) you can easily configure it via annotations.


```java
% startHighlight %
@HystrixCommand(fallbackMethod = "reliable")
% endHighlight %
public String readingList() {
	URI uri = URI.create("${target.application.http-outgoing-calls[0]}....");
	return this.restTemplate.getForObject(uri, String.class);
	}
% startHighlight %
public String reliable() {
	return "Steadybit";
}
% endHighlight %
```
