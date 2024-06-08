GET http://caddylura:8082/__custom/echo/baz/

{
    "foo": "bar"
}

HTTP 200

[Asserts]
jsonpath "$.req_uri" == "/__custom/echo/baz/"
jsonpath "$.req_body" contains "bar"
