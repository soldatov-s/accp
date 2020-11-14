# accp
Access Control Caching Proxy

## Purposes
The main problem of caching http proxy is an invalidation of cache.
Nginx invalidates cache by deleting it when expire it's TTL or by HTTP PURGE method in requests or by manual deleting cache.
In some case this approach is not enouth. It is not necessary delete cache, more effective update cache
by some conditions. For example refreshing cache by TTL or by count requests.

## Architecture
![accp-arhitecture](http://www.plantuml.com/plantuml/proxy?cache=no&src=https://raw.githubusercontent.com/soldatov-s/accp/alfa/doc/accp.puml?token=ALAUH2DWQLAWV54PN5ECZBK7WAKOA)