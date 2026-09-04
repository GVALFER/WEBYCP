import { Button, Input, Label, TextField, toast } from "@heroui/react";
import { LockKeyhole, RefreshCw } from "lucide-react";
import { type FormEvent, useState } from "react";
import useSWR, { useSWRConfig } from "swr";
import * as v from "valibot";

import type {
  AuthResponse,
  CertificateJobResponse,
  CertificateListResponse,
  DomainListResponse,
} from "../../api/types";
import { SelectField } from "../../components/SelectField";
import { api, fetcher } from "../../lib/api";
import { cn } from "../../utils/classnames";
import { formatDate } from "../../utils/date";
import { errorMessage } from "../../utils/errors";
import { statusClass } from "../../utils/status";
import {
  domainField,
  emailField,
  issueMessage,
} from "../../utils/validation";

export const CertificatesPanel = ({ session }: { session: AuthResponse }) => {
  const [domainId, setDomainId] = useState("");
  const [email, setEmail] = useState(session.user.email);
  const [hostname, setHostname] = useState("");
  const [pending, setPending] = useState("");
  const { mutate: mutateKey } = useSWRConfig();
  const { data: domains } = useSWR<DomainListResponse>("domains", fetcher);
  const { data, mutate } =
    useSWR<CertificateListResponse>("certificates", fetcher);
  const eligibleDomains =
    domains?.items.filter(
      (domain) => domain.status === "active" && domain.enabled,
    ) ?? [];
  const selectedDomain = domainId || eligibleDomains[0]?.id || "";

  const run = async (
    key: string,
    action: () => Promise<unknown>,
    success: string,
  ) => {
    setPending(key);
    try {
      await action();
      await Promise.all([mutate(), mutateKey("jobs")]);
      toast.success(success);
    } catch (error) {
      toast.danger("Certificate action failed", {
        description: await errorMessage(error),
      });
    } finally {
      setPending("");
    }
  };

  const submitDomain = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const result = v.safeParse(emailField, email);
    if (!result.success) {
      toast.warning("Check the ACME email", {
        description: issueMessage(result.issues),
      });
      return;
    }
    if (!selectedDomain) return;
    await run(
      "domain",
      () =>
        api
          .post(`domains/${encodeURIComponent(selectedDomain)}/certificate`, {
            json: { email: result.output },
          })
          .json<CertificateJobResponse>(),
      "Certificate request queued",
    );
  };

  const submitPanel = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const result = v.safeParse(
      v.object({ hostname: domainField, email: emailField }),
      { hostname, email },
    );
    if (!result.success) {
      toast.warning("Check the certificate details", {
        description: issueMessage(result.issues),
      });
      return;
    }
    await run(
      "panel",
      () =>
        api
          .post("certificates/panel", { json: result.output })
          .json<CertificateJobResponse>(),
      "Panel certificate request queued",
    );
  };

  return (
    <div className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_22rem]">
      <section className="panel-card overflow-hidden">
        <div className="border-b border-divider px-6 py-5">
          <h2 className="text-base font-semibold">TLS certificates</h2>
          <div className="mt-1 text-sm text-foreground-500">
            Let&apos;s Encrypt certificates, expiry and HTTPS redirect policy.
          </div>
        </div>
        <div className="divide-y divide-divider">
          {data?.items.length ? (
            data.items.map((certificate) => (
              <div key={certificate.id} className="px-6 py-5">
                <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
                  <div className="flex min-w-0 items-center gap-4">
                    <div className="icon-box">
                      <LockKeyhole className="size-5" />
                    </div>
                    <div className="min-w-0">
                      <div className="truncate font-medium">
                        {certificate.name}
                      </div>
                      <div className="mt-1 text-xs text-foreground-400">
                        {certificate.kind} · expires {formatDate(certificate.expiresAt)}
                      </div>
                    </div>
                  </div>
                  <div className="flex flex-wrap items-center gap-2">
                    <span
                      className={cn(
                        "rounded-full px-2.5 py-1 text-xs capitalize",
                        statusClass(certificate.status),
                      )}
                    >
                      {certificate.status}
                    </span>
                    {certificate.kind === "domain" && (
                      <Button
                        size="sm"
                        variant="tertiary"
                        isDisabled={
                          certificate.status === "pending" ||
                          pending === certificate.id
                        }
                        onPress={() =>
                          void run(
                            certificate.id,
                            () =>
                              api
                                .patch(
                                  `certificates/${encodeURIComponent(certificate.id)}`,
                                  {
                                    json: {
                                      redirectHttps: !certificate.redirectHttps,
                                    },
                                  },
                                )
                                .json<CertificateJobResponse>(),
                            certificate.redirectHttps
                              ? "HTTPS redirect disabled"
                              : "HTTPS redirect enabled",
                          )
                        }
                      >
                        {certificate.redirectHttps
                          ? "Disable redirect"
                          : "Enable redirect"}
                      </Button>
                    )}
                    <Button
                      isIconOnly
                      size="sm"
                      variant="secondary"
                      aria-label={`Renew ${certificate.name}`}
                      isDisabled={
                        certificate.status === "pending" ||
                        pending === certificate.id
                      }
                      onPress={() =>
                        void run(
                          certificate.id,
                          () =>
                            api
                              .post(
                                `certificates/${encodeURIComponent(certificate.id)}/renew`,
                              )
                              .json<CertificateJobResponse>(),
                          "Certificate renewal queued",
                        )
                      }
                    >
                      <RefreshCw className="size-4" />
                    </Button>
                  </div>
                </div>
                {certificate.names.length > 1 && (
                  <div className="mt-3 text-xs text-foreground-500">
                    SANs: {certificate.names.join(", ")}
                  </div>
                )}
                {certificate.error && (
                  <div className="mt-3 text-sm text-danger">
                    {certificate.error}
                  </div>
                )}
              </div>
            ))
          ) : (
            <div className="px-6 py-12 text-center text-sm text-foreground-400">
              No certificates yet. HTTP remains available for ACME bootstrap.
            </div>
          )}
        </div>
      </section>

      <aside className="space-y-6">
        <form className="panel-card p-6" onSubmit={submitDomain}>
          <h2 className="text-base font-semibold">Secure a domain</h2>
          <SelectField
            className="mt-5"
            label="Domain"
            value={selectedDomain}
            options={eligibleDomains}
            onChange={setDomainId}
          />
          <TextField className="mt-4" fullWidth isRequired>
            <Label>ACME email</Label>
            <Input
              type="email"
              value={email}
              onChange={(event) => setEmail(event.currentTarget.value)}
            />
          </TextField>
          <Button
            className="mt-5"
            type="submit"
            variant="primary"
            fullWidth
            isDisabled={!selectedDomain || pending !== ""}
          >
            Issue certificate
          </Button>
        </form>

        {session.user.role === "admin" && (
          <form className="panel-card p-6" onSubmit={submitPanel}>
            <h2 className="text-base font-semibold">Panel certificate</h2>
            <div className="mt-1 text-sm text-foreground-500">
              Use after the hostname resolves to this server.
            </div>
            <TextField className="mt-5" fullWidth isRequired>
              <Label>Panel hostname</Label>
              <Input
                placeholder="panel.example.com"
                value={hostname}
                onChange={(event) => setHostname(event.currentTarget.value)}
              />
            </TextField>
            <Button
              className="mt-5"
              type="submit"
              variant="secondary"
              fullWidth
              isDisabled={pending !== ""}
            >
              Secure panel
            </Button>
          </form>
        )}
      </aside>
    </div>
  );
};
