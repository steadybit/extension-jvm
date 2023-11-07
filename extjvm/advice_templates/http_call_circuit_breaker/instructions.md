Consider configuring the timeout for ${target.application.name} globally in the Spring Container. Adding an appropriate timeout as well as a fallback helps to improve by decoupling ${target.application.name}.

```java

@Bean
public RestTemplateBuilder restTemplateBuilder(RestTemplateBuilderConfigurer configurer)
	{
	return configurer.configure(new RestTemplateBuilder())
	% startHighlight %
	.setConnectTimeout(Duration.ofSeconds(5))
	.setReadTimeout(Duration.ofSeconds(2));
% endHighlight %
	}
```
