import type { Institution, InstitutionListFilter } from "./api";

export type InstitutionRow = {
  id: string;
  name: string;
  type: string;
  countryCode: string;
  status: string;
};

export function buildInstitutionsPath(filter: InstitutionListFilter = {}): string {
  const params = new URLSearchParams();

  if (filter.limit) {
    params.set("limit", String(filter.limit));
  }

  if (filter.status) {
    params.set("status", filter.status);
  }

  if (filter.page) {
    params.set("page", String(filter.page));
  }

  if (filter.pageSize) {
    params.set("pageSize", String(filter.pageSize));
  }

  const query = params.toString();
  return query ? `/v1/institutions?${query}` : "/v1/institutions";
}

export function toInstitutionRow(institution: Institution): InstitutionRow {
  return {
    id: institution.id,
    name: institution.name,
    type: formatInstitutionType(institution.type),
    countryCode: institution.countryCode,
    status: formatInstitutionStatus(institution.status)
  };
}

function formatInstitutionType(type: string): string {
  return type.charAt(0).toUpperCase() + type.slice(1).toLowerCase();
}

function formatInstitutionStatus(status: string): string {
  return status.charAt(0).toUpperCase() + status.slice(1).toLowerCase();
}
