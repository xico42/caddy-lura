GET http://{{host}}/__debug/any/

HTTP 200

[Asserts]
jsonpath "$.message" == "pong"
