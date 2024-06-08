GET http://caddylura:8082/__debug/any/

HTTP 200

[Asserts]
jsonpath "$.message" == "pong"
