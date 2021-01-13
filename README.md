# accp
Access Control Caching Proxy

## Purposes
The main problem of caching http proxy is an invalidation of cache.
Nginx invalidates cache by deleting it when expire it's TTL or by HTTP PURGE method in requests or by manual deleting cache.
In some case this approach is not enouth. It is not necessary delete cache, more effective update cache
by some conditions. For example periodical refreshing cache or by count requests.

## Features
* introspection of access tokens and embedding their content to the request body;
* limitig the number of requests to the backend;
* caching of responses;
* periodical refreshing cache or by count requests;
* sending events to queue for business analitics.

### Limitig the number of requests
Multiple clients asynchronous send identical requests to ACCP, it's reorganizes them in sequenced requests.
ACCP passes to back service only first request and all next requests wait answer from first request.

## Architecture
![accp-arhitecture](http://www.plantuml.com/plantuml/proxy?cache=no&src=https://raw.githubusercontent.com/soldatov-s/accp/alfa/doc/accp.puml)

## Sequence
![accp-sequence](http://www.plantuml.com/plantuml/proxy?cache=no&src=https://raw.githubusercontent.com/soldatov-s/accp/alfa/doc/accp-sequence.puml)

  