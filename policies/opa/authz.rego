package opentreasury.authz

# Authorization for the core API. Input shape:
# {
#   "principal": {"roles": [...], "institutionId": "..."},
#   "action": "read" | "write",
#   "resource": "transactions" | "journal" | "balances" | "accounts" |
#               "institutions" | "audit-events",
#   "institutionId": "<institution the request targets, or empty>"
# }
#
# Default deny: nothing is allowed unless a rule permits it.

default allow := false

# Treasury administrators have full cross-institution access.
allow if {
	some role in input.principal.roles
	role == "treasury-admin"
}

# Auditors have read-only cross-institution access.
allow if {
	input.action == "read"
	some role in input.principal.roles
	role == "auditor"
}

# Institution users may act only within their own institution. The chart of
# accounts is shared reference data, so it is readable regardless of scope.
allow if {
	input.action == "read"
	input.resource == "accounts"
	some role in input.principal.roles
	role == "institution-user"
}

allow if {
	some role in input.principal.roles
	role == "institution-user"
	institution_scoped
}

institution_scoped if {
	input.institutionId == ""
	input.action == "read"
}

institution_scoped if {
	input.institutionId != ""
	input.institutionId == input.principal.institutionId
}
