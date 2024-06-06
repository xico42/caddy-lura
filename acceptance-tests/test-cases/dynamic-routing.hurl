GET http://caddylura:8082/dynamic
X-Tenant-Id: 42

HTTP 200

[Asserts]
jsonpath "$.tenant_id" == 42

GET http://caddylura:8082/dynamic
X-Tenant-Id: 21

HTTP 200

[Asserts]
jsonpath "$.tenant_id" == 21
