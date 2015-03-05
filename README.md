# Micro MQ

A simplistic pub/sub message queue for testing

## Pub

```
curl -d "A completely arbitrary message" "http://localhost:8081/pub?topic=foo"
```

## Sub

```
curl -i -N -H "Connection: Upgrade" -H "Upgrade: websocket" -H "Host: localhost:8081" -H "Origin:http://localhost:8081" "http://localhost:8081/sub?topic=foo"
```
