import { Button, Input, Label, TextField } from "@heroui/react";
import { LockKeyhole, RefreshCw } from "lucide-react";
import { type FormEvent, useState } from "react";
import useSWR, { useSWRConfig } from "swr";

import {
  fetcher,
  issueDomainCertificate,
  issuePanelCertificate,
  renewCertificate,
  setCertificate,
  type AuthResponse,
  type CertificateListResponse,
  type DomainListResponse,
} from "../../lib/api";
import { cn } from "../../utils/classnames";
import { formatDate } from "../../utils/date";
import { errorMessage } from "../../utils/errors";
import { statusClass } from "../../utils/status";

export const CertificatesPanel = ({ session }: { session: AuthResponse }) => {
  const [domainId, setDomainId] = useState("");
  const [email, setEmail] = useState(session.user.email);
  const [hostname, setHostname] = useState("");
  const [pending, setPending] = useState("");
  const [error, setError] = useState("");
  const { mutate: mutateKey } = useSWRConfig();
  const { data: domains } = useSWR<DomainListResponse>("domains", fetcher);
  const { data, mutate } = useSWR<CertificateListResponse>("certificates", fetcher, { refreshInterval: 5_000 });
  const eligibleDomains = domains?.items.filter((domain) => domain.status === "active" && domain.enabled) ?? [];
  const selectedDomain = domainId || eligibleDomains[0]?.id || "";

  const run = async (key: string, action: () => Promise<unknown>) => {
    setPending(key);
    setError("");
    try {
      await action();
      await Promise.all([mutate(), mutateKey("jobs")]);
    } catch (requestError) {
      setError(await errorMessage(requestError));
    } finally {
      setPending("");
    }
  };

  const submitDomain = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (selectedDomain) await run("domain", () => issueDomainCertificate(selectedDomain, email, session.csrfToken));
  };

  const submitPanel = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    await run("panel", () => issuePanelCertificate(hostname, email, session.csrfToken));
  };

  return (
    <div className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_22rem]">
      <section className="overflow-hidden rounded-2xl border border-divider bg-surface">
        <div className="border-b border-divider px-6 py-5"><h2 className="text-lg font-semibold">TLS certificates</h2><div className="mt-1 text-sm text-foreground-500">Let&apos;s Encrypt certificates, eligible SANs, expiry and redirect policy.</div>{error && <div className="mt-4 rounded-xl border border-danger/30 bg-danger/10 px-4 py-3 text-sm text-danger">{error}</div>}</div>
        <div className="divide-y divide-divider">
          {data?.items.length ? data.items.map((certificate) => <div key={certificate.id} className="px-6 py-5">
            <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
              <div className="flex items-center gap-4"><div className="flex size-10 items-center justify-center rounded-xl bg-default/10"><LockKeyhole className="size-5" /></div><div><div className="font-medium">{certificate.name}</div><div className="mt-1 text-xs text-foreground-400">{certificate.kind} · expires {formatDate(certificate.expiresAt)}</div></div></div>
              <div className="flex items-center gap-2"><span className={cn("rounded-full px-2.5 py-1 text-xs", statusClass(certificate.status))}>{certificate.status}</span>{certificate.kind === "domain" && <Button size="sm" variant="tertiary" isDisabled={certificate.status === "pending" || pending === certificate.id} onPress={() => void run(certificate.id, () => setCertificate(certificate.id, !certificate.redirectHttps, session.csrfToken))}>{certificate.redirectHttps ? "Disable redirect" : "Enable redirect"}</Button>}<Button isIconOnly size="sm" variant="secondary" aria-label={`Renew ${certificate.name}`} isDisabled={certificate.status === "pending" || pending === certificate.id} onPress={() => void run(certificate.id, () => renewCertificate(certificate.id, session.csrfToken))}><RefreshCw className="size-4" /></Button></div>
            </div>
            {certificate.names.length > 1 && <div className="mt-3 text-xs text-foreground-500">SANs: {certificate.names.join(", ")}</div>}
            {certificate.error && <div className="mt-3 text-sm text-danger">{certificate.error}</div>}
          </div>) : <div className="px-6 py-12 text-center text-sm text-foreground-400">No certificates yet. HTTP remains available for ACME bootstrap.</div>}
        </div>
      </section>
      <aside className="space-y-6">
        <form className="rounded-2xl border border-divider bg-surface p-6" onSubmit={submitDomain}><h2 className="text-lg font-semibold">Secure a domain</h2><label className="mt-5 block space-y-2 text-sm"><span>Domain</span><select className="h-10 w-full rounded-lg border border-divider bg-background px-3" value={selectedDomain} onChange={(event) => setDomainId(event.currentTarget.value)}>{eligibleDomains.map((domain) => <option key={domain.id} value={domain.id}>{domain.name}</option>)}</select></label><TextField className="mt-4" fullWidth isRequired><Label>ACME email</Label><Input type="email" value={email} onChange={(event) => setEmail(event.currentTarget.value)} /></TextField><Button className="mt-5" type="submit" variant="primary" fullWidth isDisabled={!selectedDomain || pending !== ""}>Issue certificate</Button></form>
        {session.user.role === "admin" && <form className="rounded-2xl border border-divider bg-surface p-6" onSubmit={submitPanel}><h2 className="text-lg font-semibold">Panel certificate</h2><div className="mt-1 text-sm text-foreground-500">Use after the hostname resolves to this server.</div><TextField className="mt-5" fullWidth isRequired><Label>Panel hostname</Label><Input placeholder="panel.example.com" value={hostname} onChange={(event) => setHostname(event.currentTarget.value)} /></TextField><Button className="mt-5" type="submit" variant="secondary" fullWidth isDisabled={pending !== ""}>Secure panel</Button></form>}
      </aside>
    </div>
  );
};
