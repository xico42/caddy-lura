GET http://{{host}}/__custom/echo/baz/

{
    "foo": "bar"
}

HTTP 200

[Asserts]
jsonpath "$.req_uri" == "/__custom/echo/baz/"
jsonpath "$.req_body" contains "bar"
