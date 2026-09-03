import ft from "reqly-js";

import type { components } from "../api/schema";

export type HealthResponse = components["schemas"]["HealthResponse"];
export type BootstrapResponse = components["schemas"]["BootstrapResponse"];
export type BootstrapRequest = components["schemas"]["BootstrapRequest"];
export type LoginRequest = components["schemas"]["LoginRequest"];
export type AuthResponse = components["schemas"]["AuthResponse"];
export type Account = components["schemas"]["Account"];
export type AccountListResponse = components["schemas"]["AccountListResponse"];
export type AccountJobResponse = components["schemas"]["AccountJobResponse"];
export type CreateAccountRequest = components["schemas"]["CreateAccountRequest"];
export type UpdateEnabledRequest = components["schemas"]["UpdateEnabledRequest"];
export type Domain = components["schemas"]["Domain"];
export type DomainListResponse = components["schemas"]["DomainListResponse"];
export type DomainJobResponse = components["schemas"]["DomainJobResponse"];
export type CreateDomainRequest = components["schemas"]["CreateDomainRequest"];
export type DomainAlias = components["schemas"]["DomainAlias"];
export type DomainAliasListResponse =
  components["schemas"]["DomainAliasListResponse"];
export type DomainAliasJobResponse =
  components["schemas"]["DomainAliasJobResponse"];
export type CreateDomainAliasRequest =
  components["schemas"]["CreateDomainAliasRequest"];
export type UpdateDomainRequest = components["schemas"]["UpdateDomainRequest"];
export type Node = components["schemas"]["Node"];
export type NodeListResponse = components["schemas"]["NodeListResponse"];
export type Job = components["schemas"]["Job"];
export type JobListResponse = components["schemas"]["JobListResponse"];
export type Certificate = components["schemas"]["Certificate"];
export type CertificateListResponse = components["schemas"]["CertificateListResponse"];
export type CertificateJobResponse = components["schemas"]["CertificateJobResponse"];
export type Database = components["schemas"]["Database"];
export type DatabaseUser = components["schemas"]["DatabaseUser"];
export type DatabaseGrant = components["schemas"]["DatabaseGrant"];
export type DatabaseListResponse = components["schemas"]["DatabaseListResponse"];
export type DatabaseUserListResponse = components["schemas"]["DatabaseUserListResponse"];
export type DatabaseGrantListResponse = components["schemas"]["DatabaseGrantListResponse"];
export type DatabaseUserJobResponse = components["schemas"]["DatabaseUserJobResponse"];
export type CronJob = components["schemas"]["CronJob"];
export type CronJobListResponse = components["schemas"]["CronJobListResponse"];
export type WriteCronJobRequest = components["schemas"]["WriteCronJobRequest"];
export type BackupPlan = components["schemas"]["BackupPlan"];
export type BackupRun = components["schemas"]["BackupRun"];
export type BackupArtifact = components["schemas"]["BackupArtifact"];
export type BackupPlanListResponse = components["schemas"]["BackupPlanListResponse"];
export type BackupRunListResponse = components["schemas"]["BackupRunListResponse"];
export type BackupArtifactListResponse = components["schemas"]["BackupArtifactListResponse"];
export type BackupManifest = components["schemas"]["BackupManifest"];
export type WriteBackupPlanRequest = components["schemas"]["WriteBackupPlanRequest"];
export type RestoreBackupRequest = components["schemas"]["RestoreBackupRequest"];

export const api = ft.create({
  baseUrl: "/",
  prefix: "api/v1",
  headers: { accept: "application/json" },
  timeout: 10_000,
});

export const fetcher = <T>(path: string) => api.get(path).json<T>();

export const createBootstrap = (data: BootstrapRequest) =>
  api.post("bootstrap", { json: data }).json<AuthResponse>();

export const login = (data: LoginRequest) =>
  api.post("auth/login", { json: data }).json<AuthResponse>();

export const logout = (csrfToken: string) =>
  api.post("auth/logout", { headers: { "X-CSRF-Token": csrfToken } });

export const createAccount = (data: CreateAccountRequest, csrfToken: string) =>
  api
    .post("accounts", {
      json: data,
      headers: { "X-CSRF-Token": csrfToken },
    })
    .json<AccountJobResponse>();

export const setAccount = (accountId: string, enabled: boolean, csrfToken: string) =>
  api.patch(`accounts/${encodeURIComponent(accountId)}`, { json: { enabled }, headers: { "X-CSRF-Token": csrfToken } }).json<AccountJobResponse>();

export const deleteAccount = (accountId: string, csrfToken: string) =>
  api.delete(`accounts/${encodeURIComponent(accountId)}`, { headers: { "X-CSRF-Token": csrfToken } }).json<AccountJobResponse>();

export const createDomain = (data: CreateDomainRequest, csrfToken: string) =>
  api
    .post("domains", {
      json: data,
      headers: { "X-CSRF-Token": csrfToken },
    })
    .json<DomainJobResponse>();

export const createDomainAlias = (
  domainId: string,
  data: CreateDomainAliasRequest,
  csrfToken: string,
) =>
  api
    .post(`domains/${encodeURIComponent(domainId)}/aliases`, {
      json: data,
      headers: { "X-CSRF-Token": csrfToken },
    })
    .json<DomainAliasJobResponse>();

const updateResource = <T>(
  path: string,
  data: unknown,
  csrfToken: string,
) =>
  api
    .patch(path, {
      json: data,
      headers: { "X-CSRF-Token": csrfToken },
    })
    .json<T>();

const remove = <T>(path: string, csrfToken: string) =>
  api
    .delete(path, { headers: { "X-CSRF-Token": csrfToken } })
    .json<T>();

export const setDomain = (
  domainId: string,
  enabled: boolean,
  csrfToken: string,
) =>
  updateResource<DomainJobResponse>(
    `domains/${encodeURIComponent(domainId)}`,
    { enabled },
    csrfToken,
  );

export const deleteDomain = (domainId: string, csrfToken: string) =>
  remove<DomainJobResponse>(
    `domains/${encodeURIComponent(domainId)}`,
    csrfToken,
  );

export const renameDomain = (
  domainId: string,
  name: string,
  csrfToken: string,
) =>
  updateResource<DomainJobResponse>(
    `domains/${encodeURIComponent(domainId)}`,
    { name },
    csrfToken,
  );

export const setDomainAlias = (
  domainId: string,
  aliasId: string,
  enabled: boolean,
  csrfToken: string,
) =>
  updateResource<DomainAliasJobResponse>(
    `domains/${encodeURIComponent(domainId)}/aliases/${encodeURIComponent(aliasId)}`,
    { enabled },
    csrfToken,
  );

export const deleteDomainAlias = (
  domainId: string,
  aliasId: string,
  csrfToken: string,
) =>
  remove<DomainAliasJobResponse>(
    `domains/${encodeURIComponent(domainId)}/aliases/${encodeURIComponent(aliasId)}`,
    csrfToken,
  );

export const renameDomainAlias = (
  domainId: string,
  aliasId: string,
  name: string,
  csrfToken: string,
) =>
  updateResource<DomainAliasJobResponse>(
    `domains/${encodeURIComponent(domainId)}/aliases/${encodeURIComponent(aliasId)}`,
    { name },
    csrfToken,
  );

export const probeNode = (nodeId: string, csrfToken: string) =>
  api
    .post(`nodes/${encodeURIComponent(nodeId)}/probe`, {
      headers: { "X-CSRF-Token": csrfToken },
    })
    .json<Job>();

export const issueDomainCertificate = (domainId: string, email: string, csrfToken: string) =>
  api.post(`domains/${encodeURIComponent(domainId)}/certificate`, { json: { email }, headers: { "X-CSRF-Token": csrfToken } }).json<CertificateJobResponse>();

export const issuePanelCertificate = (hostname: string, email: string, csrfToken: string) =>
  api.post("certificates/panel", { json: { hostname, email }, headers: { "X-CSRF-Token": csrfToken } }).json<CertificateJobResponse>();

export const renewCertificate = (id: string, csrfToken: string) =>
  api.post(`certificates/${encodeURIComponent(id)}/renew`, { headers: { "X-CSRF-Token": csrfToken } }).json<CertificateJobResponse>();

export const setCertificate = (id: string, redirectHttps: boolean, csrfToken: string) =>
  updateResource<CertificateJobResponse>(`certificates/${encodeURIComponent(id)}`, { redirectHttps }, csrfToken);

export const createDatabase = (accountId: string, name: string, csrfToken: string) =>
  api.post("databases", { json: { accountId, name }, headers: { "X-CSRF-Token": csrfToken } }).json<components["schemas"]["DatabaseJobResponse"]>();

export const deleteDatabase = (id: string, csrfToken: string) =>
  remove<components["schemas"]["DatabaseJobResponse"]>(`databases/${encodeURIComponent(id)}`, csrfToken);

export const createDatabaseUser = (accountId: string, name: string, csrfToken: string) =>
  api.post("database-users", { json: { accountId, name }, headers: { "X-CSRF-Token": csrfToken } }).json<DatabaseUserJobResponse>();

export const deleteDatabaseUser = (id: string, csrfToken: string) =>
  remove<DatabaseUserJobResponse>(`database-users/${encodeURIComponent(id)}`, csrfToken);

export const setDatabaseGrant = (databaseId: string, userId: string, enabled: boolean, csrfToken: string) => {
  const path = `databases/${encodeURIComponent(databaseId)}/users/${encodeURIComponent(userId)}`;
  const options = { headers: { "X-CSRF-Token": csrfToken } };
  return (enabled ? api.put(path, options) : api.delete(path, options)).json<components["schemas"]["DatabaseGrantJobResponse"]>();
};

export const createCronJob = (data: WriteCronJobRequest, csrfToken: string) =>
  api.post("cron-jobs", { json: data, headers: { "X-CSRF-Token": csrfToken } }).json<components["schemas"]["CronJobResponse"]>();

export const updateCronJob = (id: string, data: WriteCronJobRequest, csrfToken: string) =>
  updateResource<components["schemas"]["CronJobResponse"]>(`cron-jobs/${encodeURIComponent(id)}`, data, csrfToken);

export const deleteCronJob = (id: string, csrfToken: string) =>
  remove<Job>(`cron-jobs/${encodeURIComponent(id)}`, csrfToken);

export const createBackupPlan = (data: WriteBackupPlanRequest, csrfToken: string) =>
  api.post("backup-plans", { json: data, headers: { "X-CSRF-Token": csrfToken } }).json<BackupPlan>();

export const updateBackupPlan = (id: string, data: WriteBackupPlanRequest, csrfToken: string) =>
  updateResource<BackupPlan>(`backup-plans/${encodeURIComponent(id)}`, data, csrfToken);

export const deleteBackupPlan = async (id: string, csrfToken: string) => {
  await api.delete(`backup-plans/${encodeURIComponent(id)}`, { headers: { "X-CSRF-Token": csrfToken } });
};

export const runBackupPlan = (id: string, csrfToken: string) =>
  api.post(`backup-plans/${encodeURIComponent(id)}/runs`, { headers: { "X-CSRF-Token": csrfToken } }).json<components["schemas"]["BackupRunResponse"]>();

export const previewBackup = (id: string) =>
  api.get(`backup-artifacts/${encodeURIComponent(id)}/restore`).json<BackupManifest>();

export const restoreBackup = (id: string, data: RestoreBackupRequest, csrfToken: string) =>
  api.post(`backup-artifacts/${encodeURIComponent(id)}/restore`, { json: data, headers: { "X-CSRF-Token": csrfToken } }).json<Job>();

export const deleteBackupArtifact = async (id: string, csrfToken: string) => {
  await api.delete(`backup-artifacts/${encodeURIComponent(id)}`, {
    headers: { "X-CSRF-Token": csrfToken },
  });
};
