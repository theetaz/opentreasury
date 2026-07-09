package opentreasury.publication

# Publication policy: which fields of each resource may appear on the
# anonymous public tier (/public/v1). Publishing data publicly is a
# security-sensitive operation — fields NOT listed here never leave the
# operator plane, and the API applies this map as a hard server-side mask.
#
# v1 stance: institutions, the chart of accounts, aggregate balances, and
# journal structure (amounts, accounts, dates) are PUBLIC. Free-text entry
# descriptions are INSTITUTION-classified by default because they may carry
# personal data (payee names); a per-deployment field review can widen this.

fields := {
	"institutions": {"id", "name", "type", "countryCode", "status"},
	"accounts": {"code", "name", "accountType", "parentCode", "gfsmCode", "cofogCode", "active", "depth"},
	"balances": {"institutionId", "accountCode", "accountName", "accountType", "currency", "balanceMinor"},
	"journal-entries": {
		"id",
		"institutionId",
		"fiscalYear",
		"effectiveDate",
		"status",
		"entryType",
		"reversesEntryId",
		"lines",
	},
}

# Line-level fields inside a public journal entry.
line_fields := {"accountCode", "direction", "amountMinor", "currency"}
