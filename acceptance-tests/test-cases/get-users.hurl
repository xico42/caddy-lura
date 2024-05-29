GET http://lura:8080/users/42

HTTP 200

[Asserts]
jsonpath "$.id" == 42
jsonpath "$.name" == "John Doe"
jsonpath "$.personal_email" == "john@bar.com"

jsonpath "$.permissions.posts" count == 1
jsonpath "$.permissions.posts[0]" == "view"
jsonpath "$.permissions.profile" count == 1
jsonpath "$.permissions.profile[0]" == "manage"
