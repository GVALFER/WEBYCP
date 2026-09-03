package certificates

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/GVALFER/WEBYCP/internal/accounts"
	"github.com/GVALFER/WEBYCP/internal/domains"
	"github.com/GVALFER/WEBYCP/internal/idgen"
	"github.com/GVALFER/WEBYCP/internal/jobs"
	"github.com/GVALFER/WEBYCP/internal/nodes"
	"github.com/GVALFER/WEBYCP/internal/validate"
)

type Service struct {
	repository Repository
	domains    *domains.Service
	accounts   *accounts.Service
	nodes      nodes.Repository
	agent      Agent
	notify     func()
}

type payload struct {
	CertificateID string `json:"certificateId"`
}

func NewService(repository Repository, domains *domains.Service, accounts *accounts.Service, nodes nodes.Repository, agent Agent, notify func()) *Service {
	return &Service{repository: repository, domains: domains, accounts: accounts, nodes: nodes, agent: agent, notify: notify}
}

func (s *Service) Certificates(ctx context.Context, userID string, admin bool) ([]Certificate, error) {
	return s.repository.Certificates(ctx, userID, admin)
}

func (s *Service) IssueDomain(ctx context.Context, domainID, email, userID string, admin bool) (Certificate, jobs.Job, error) {
	domain, err := s.domains.GetDomain(ctx, domainID, userID, admin)
	if err != nil {
		return Certificate{}, jobs.Job{}, err
	}
	if domain.Status != "active" || !domain.Enabled {
		return Certificate{}, jobs.Job{}, domains.ErrDomainInactive
	}
	email, err = validate.Email(email)
	if err != nil {
		return Certificate{}, jobs.Job{}, err
	}
	aliases, err := s.domains.Aliases(ctx, domain.ID, userID, admin)
	if err != nil {
		return Certificate{}, jobs.Job{}, err
	}
	names := []string{domain.Name}
	for _, alias := range aliases {
		if alias.Enabled && alias.Status == "active" {
			names = append(names, alias.Name)
		}
	}
	certificate, create, err := s.domainValue(ctx, domain, email, names)
	if err != nil {
		return Certificate{}, jobs.Job{}, err
	}
	return s.queue(ctx, certificate, userID, jobs.KindCertificateIssue, create)
}

func (s *Service) IssuePanel(ctx context.Context, hostname, email, userID string) (Certificate, jobs.Job, error) {
	hostname, err := validate.Domain(hostname)
	if err != nil {
		return Certificate{}, jobs.Job{}, err
	}
	email, err = validate.Email(email)
	if err != nil {
		return Certificate{}, jobs.Job{}, err
	}
	all, err := s.nodes.Nodes(ctx)
	if err != nil || len(all) == 0 {
		return Certificate{}, jobs.Job{}, fmt.Errorf("get panel node: %w", err)
	}
	certificate, err := s.repository.PanelCertificate(ctx)
	create := errors.Is(err, sql.ErrNoRows)
	if err != nil && !create {
		return Certificate{}, jobs.Job{}, err
	}
	if create {
		id, err := idgen.ID()
		if err != nil {
			return Certificate{}, jobs.Job{}, err
		}
		now := time.Now().UTC()
		certificate = Certificate{ID: id, NodeID: all[0].ID, Kind: "panel", Status: "pending", RedirectHTTPS: true, CreatedAt: now, UpdatedAt: now}
	} else if certificate.Status == "pending" {
		return Certificate{}, jobs.Job{}, ErrBusy
	}
	certificate.Name, certificate.Names, certificate.Email = hostname, []string{hostname}, email
	return s.queue(ctx, certificate, userID, jobs.KindCertificateIssue, create)
}

func (s *Service) Renew(ctx context.Context, id, userID string, admin bool) (Certificate, jobs.Job, error) {
	certificate, err := s.access(ctx, id, userID, admin)
	if err != nil {
		return Certificate{}, jobs.Job{}, err
	}
	if certificate.Status == "pending" {
		return Certificate{}, jobs.Job{}, ErrBusy
	}
	return s.queue(ctx, certificate, userID, jobs.KindCertificateRenew, false)
}

func (s *Service) SetRedirect(ctx context.Context, id, userID string, admin, redirect bool) (Certificate, jobs.Job, error) {
	certificate, err := s.access(ctx, id, userID, admin)
	if err != nil {
		return Certificate{}, jobs.Job{}, err
	}
	if certificate.Kind != "domain" || certificate.Status == "pending" {
		return Certificate{}, jobs.Job{}, ErrBusy
	}
	certificate.RedirectHTTPS = redirect
	return s.queue(ctx, certificate, userID, jobs.KindCertificateRenew, false)
}

func (s *Service) QueueDue(ctx context.Context) error {
	values, err := s.repository.DueCertificates(ctx, time.Now().UTC())
	if err != nil {
		return err
	}
	for _, value := range values {
		pending, err := s.repository.CertificateJobPending(ctx, value.ID)
		if err != nil {
			return err
		}
		if pending {
			continue
		}
		if _, _, err := s.queue(ctx, value, "", jobs.KindCertificateRenew, false); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) Provision(ctx context.Context, job jobs.Job) error {
	var value payload
	if err := json.Unmarshal([]byte(job.Payload), &value); err != nil || value.CertificateID == "" {
		return fmt.Errorf("decode certificate job payload")
	}
	certificate, err := s.repository.Certificate(ctx, value.CertificateID)
	if err != nil {
		return err
	}
	return s.provision(ctx, certificate)
}

func (s *Service) ReconcileDomain(ctx context.Context, domainID string) error {
	certificate, err := s.repository.DomainCertificate(ctx, domainID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if certificate.Status != "active" {
		return nil
	}
	return s.provision(ctx, certificate)
}

func (s *Service) provision(ctx context.Context, certificate Certificate) error {
	node, err := s.nodes.Node(ctx, certificate.NodeID)
	if err != nil {
		return err
	}
	request := Request{CertificateID: certificate.ID, Kind: certificate.Kind, DomainID: certificate.DomainID, Name: certificate.Name, Names: certificate.Names, Email: certificate.Email, RedirectHTTPS: certificate.RedirectHTTPS}
	if certificate.Kind == "domain" {
		domain, err := s.domains.Get(ctx, certificate.DomainID)
		if err != nil {
			return err
		}
		account, err := s.accounts.Get(ctx, domain.AccountID)
		if err != nil {
			return err
		}
		request.AccountID, request.SystemUser, request.PHPVersion = account.ID, account.SystemUser, domain.PHPVersion
	}
	result, err := s.agent.IssueCertificate(ctx, node.Endpoint, request)
	if err != nil {
		_ = s.repository.SetResult(ctx, certificate.ID, certificate.Names, "error", nil, nil, safeError(err))
		return err
	}
	renewAfter := result.ExpiresAt.Add(-30 * 24 * time.Hour)
	return s.repository.SetResult(ctx, certificate.ID, result.Names, "active", &result.ExpiresAt, &renewAfter, "")
}

func (s *Service) domainValue(ctx context.Context, domain domains.Domain, email string, names []string) (Certificate, bool, error) {
	certificate, err := s.repository.DomainCertificate(ctx, domain.ID)
	create := errors.Is(err, sql.ErrNoRows)
	if err != nil && !create {
		return Certificate{}, false, err
	}
	if create {
		id, err := idgen.ID()
		if err != nil {
			return Certificate{}, false, err
		}
		now := time.Now().UTC()
		certificate = Certificate{ID: id, DomainID: domain.ID, NodeID: domain.NodeID, Kind: "domain", Status: "pending", RedirectHTTPS: true, CreatedAt: now, UpdatedAt: now}
	} else if certificate.Status == "pending" {
		return Certificate{}, false, ErrBusy
	}
	certificate.Name, certificate.Names, certificate.Email = domain.Name, names, email
	return certificate, create, nil
}

func (s *Service) access(ctx context.Context, id, userID string, admin bool) (Certificate, error) {
	if err := validate.ID("certificateId", id); err != nil {
		return Certificate{}, err
	}
	certificate, err := s.repository.Certificate(ctx, id)
	if err != nil {
		return Certificate{}, err
	}
	if certificate.Kind == "panel" {
		if !admin {
			return Certificate{}, accounts.ErrForbidden
		}
		return certificate, nil
	}
	if _, err := s.domains.GetDomain(ctx, certificate.DomainID, userID, admin); err != nil {
		return Certificate{}, err
	}
	return certificate, nil
}

func (s *Service) queue(ctx context.Context, certificate Certificate, userID, kind string, create bool) (Certificate, jobs.Job, error) {
	id, err := idgen.ID()
	if err != nil {
		return Certificate{}, jobs.Job{}, err
	}
	data, err := json.Marshal(payload{CertificateID: certificate.ID})
	if err != nil {
		return Certificate{}, jobs.Job{}, err
	}
	job := jobs.Job{ID: id, NodeID: certificate.NodeID, UserID: userID, Kind: kind, Status: "queued", Payload: string(data), MaxAttempts: 2, CreatedAt: time.Now().UTC()}
	certificate, job, err = s.repository.QueueCertificate(ctx, certificate, job, create)
	if err == nil {
		s.notify()
	}
	return certificate, job, err
}

func safeError(err error) string {
	message := err.Error()
	if len(message) > 500 {
		return message[:500]
	}
	return message
}
