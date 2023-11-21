When ${target.application.name}'s downstream endpoints aren't working correctly, ${target.application.name} doesn't back off requesting the endpoint, which risks the downstream application becoming unavailable. Eventually, this may lead to a catastrophic cascade as ${target.application.name} isn't correctly working either, causing upstream failures.

***Downstream Endpoints***
${target.application.http-outgoing-calls[]:ul}