package websites

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/GVALFER/WEBYCP/internal/accounts"
	"github.com/GVALFER/WEBYCP/internal/idgen"
	"github.com/GVALFER/WEBYCP/internal/jobs"
	"github.com/GVALFER/WEBYCP/internal/nodes"
	"github.com/GVALFER/WEBYCP/internal/pagination"
	"github.com/GVALFER/WEBYCP/internal/services"
	"github.com/GVALFER/WEBYCP/internal/validate"
)

type Service struct {
	repository Repository
	accounts   *accounts.Service
	nodes      nodes.Repository
	agent      Agent
	notify     func()
}

type websitePayload struct {
	WebsiteID string `json:"websiteId"`
}

type domainPayload struct {
	WebsiteDomainID string `json:"websiteDomainId"`
}

type renamePayload struct {
	WebsiteDomainID  string `json:"websiteDomainId"`
	PreviousHostname string `json:"previousHostname"`
	Hostname         string `json:"hostname"`
}

func NewService(repository Repository, accountService *accounts.Service, nodeRepository nodes.Repository, agent Agent, notify func()) *Service {
	return &Service{repository: repository, accounts: accountService, nodes: nodeRepository, agent: agent, notify: notify}
}

func (s *Service) Create(ctx context.Context, value Website, primaryHostname, userID string, admin bool) (Website, WebsiteDomain, jobs.Job, error) {
	if err := validate.ID("accountId", value.AccountID); err != nil {
		return Website{}, WebsiteDomain{}, jobs.Job{}, err
	}
	name, err := validate.ResourceName(value.Name)
	if err != nil {
		return Website{}, WebsiteDomain{}, jobs.Job{}, err
	}
	hostname, err := validate.Domain(primaryHostname)
	if err != nil {
		return Website{}, WebsiteDomain{}, jobs.Job{}, err
	}
	if value.Kind != "php" || value.WebDriver != services.Nginx ||
		value.RuntimeDriver != services.PHPFPM || value.RuntimeVersion != services.PHP83 {
		return Website{}, WebsiteDomain{}, jobs.Job{}, &validate.Error{Field: "driver", Message: "The selected website stack is not supported"}
	}
	account, err := s.accounts.Account(ctx, value.AccountID, userID, admin)
	if err != nil {
		return Website{}, WebsiteDomain{}, jobs.Job{}, err
	}
	if account.Status != "active" || !account.Enabled {
		return Website{}, WebsiteDomain{}, jobs.Job{}, accounts.ErrBusy
	}
	websiteID, err := idgen.ID()
	if err != nil {
		return Website{}, WebsiteDomain{}, jobs.Job{}, err
	}
	domainID, err := idgen.ID()
	if err != nil {
		return Website{}, WebsiteDomain{}, jobs.Job{}, err
	}
	now := time.Now().UTC()
	value.ID, value.AccountID, value.NodeID = websiteID, account.ID, account.NodeID
	value.Name, value.DocumentRoot, value.Status, value.Enabled = name, filepath.Join("/home", account.SystemUser, "web", websiteID, "public_html"), "pending", true
	value.CreatedAt, value.UpdatedAt = now, now
	domain := WebsiteDomain{ID: domainID, WebsiteID: websiteID, Hostname: hostname, Kind: "primary", Status: "pending", Enabled: true, CreatedAt: now, UpdatedAt: now}
	job, err := newJob(account.NodeID, userID, jobs.KindWebsiteCreate, websitePayload{WebsiteID: websiteID})
	if err != nil {
		return Website{}, WebsiteDomain{}, jobs.Job{}, err
	}
	value, domain, job, err = s.repository.CreateWebsiteProvision(ctx, value, domain, job)
	if err != nil {
		return Website{}, WebsiteDomain{}, jobs.Job{}, fmt.Errorf("create website provision: %w", err)
	}
	s.notify()
	return value, domain, job, nil
}

func (s *Service) Websites(ctx context.Context, userID string, admin bool) ([]Website, error) {
	return s.repository.Websites(ctx, userID, admin)
}

func (s *Service) WebsitePage(ctx context.Context, userID string, admin bool, query pagination.Query) (pagination.Result[Website], error) {
	return s.repository.WebsitePage(ctx, userID, admin, query)
}

func (s *Service) Get(ctx context.Context, id string) (Website, error) {
	return s.repository.Website(ctx, id)
}

func (s *Service) GetWebsite(ctx context.Context, id, userID string, admin bool) (Website, error) {
	return s.website(ctx, id, userID, admin)
}

func (s *Service) PrimaryDomain(ctx context.Context, websiteID, userID string, admin bool) (WebsiteDomain, error) {
	if _, err := s.website(ctx, websiteID, userID, admin); err != nil {
		return WebsiteDomain{}, err
	}
	return s.repository.PrimaryDomain(ctx, websiteID)
}

func (s *Service) Domains(ctx context.Context, websiteID, userID string, admin bool) ([]WebsiteDomain, error) {
	if _, err := s.website(ctx, websiteID, userID, admin); err != nil {
		return nil, err
	}
	return s.repository.WebsiteDomains(ctx, websiteID)
}

func (s *Service) DomainPage(ctx context.Context, userID string, admin bool, kind string, query pagination.Query) (pagination.Result[WebsiteDomain], error) {
	if kind != "primary" && kind != "alias" {
		return pagination.Result[WebsiteDomain]{}, &validate.Error{Field: "kind", Message: "Website domain kind is invalid"}
	}
	return s.repository.WebsiteDomainPage(ctx, userID, admin, kind, query)
}

func (s *Service) CreateDomain(ctx context.Context, websiteID, hostname, userID string, admin bool) (WebsiteDomain, jobs.Job, error) {
	website, err := s.website(ctx, websiteID, userID, admin)
	if err != nil {
		return WebsiteDomain{}, jobs.Job{}, err
	}
	if website.Status != "active" || !website.Enabled {
		return WebsiteDomain{}, jobs.Job{}, ErrWebsiteInactive
	}
	values, err := s.repository.WebsiteDomains(ctx, website.ID)
	if err != nil {
		return WebsiteDomain{}, jobs.Job{}, err
	}
	if len(values)-1 >= validate.MaxDomainAliases {
		return WebsiteDomain{}, jobs.Job{}, &validate.Error{Field: "hostname", Message: "This website has reached its alias limit"}
	}
	hostname, err = validate.Domain(hostname)
	if err != nil {
		return WebsiteDomain{}, jobs.Job{}, err
	}
	id, err := idgen.ID()
	if err != nil {
		return WebsiteDomain{}, jobs.Job{}, err
	}
	now := time.Now().UTC()
	domain := WebsiteDomain{ID: id, WebsiteID: website.ID, Hostname: hostname, Kind: "alias", Status: "pending", Enabled: true, CreatedAt: now, UpdatedAt: now}
	job, err := newJob(website.NodeID, userID, jobs.KindWebsiteDomainCreate, domainPayload{WebsiteDomainID: id})
	if err != nil {
		return WebsiteDomain{}, jobs.Job{}, err
	}
	domain, job, err = s.repository.CreateWebsiteDomainProvision(ctx, domain, job)
	if err == nil {
		s.notify()
	}
	return domain, job, err
}

func (s *Service) SetWebsite(ctx context.Context, websiteID, userID string, admin, enabled bool) (Website, jobs.Job, error) {
	website, err := s.website(ctx, websiteID, userID, admin)
	if err != nil {
		return Website{}, jobs.Job{}, err
	}
	if website.Status == "pending" {
		return Website{}, jobs.Job{}, ErrWebsiteBusy
	}
	kind := jobs.KindWebsiteDisable
	if enabled {
		kind = jobs.KindWebsiteEnable
	}
	job, err := newJob(website.NodeID, userID, kind, websitePayload{WebsiteID: website.ID})
	if err != nil {
		return Website{}, jobs.Job{}, err
	}
	website, job, err = s.repository.QueueWebsiteAction(ctx, website.ID, enabled, job)
	if err == nil {
		s.notify()
	}
	return website, job, err
}

func (s *Service) DeleteWebsite(ctx context.Context, websiteID, userID string, admin bool) (Website, jobs.Job, error) {
	website, err := s.website(ctx, websiteID, userID, admin)
	if err != nil {
		return Website{}, jobs.Job{}, err
	}
	if website.Status == "pending" {
		return Website{}, jobs.Job{}, ErrWebsiteBusy
	}
	job, err := newJob(website.NodeID, userID, jobs.KindWebsiteDelete, websitePayload{WebsiteID: website.ID})
	if err != nil {
		return Website{}, jobs.Job{}, err
	}
	website, job, err = s.repository.QueueWebsiteAction(ctx, website.ID, false, job)
	if err == nil {
		s.notify()
	}
	return website, job, err
}

func (s *Service) SetDomain(ctx context.Context, domainID, userID string, admin, enabled bool) (WebsiteDomain, jobs.Job, error) {
	website, domain, err := s.domain(ctx, domainID, userID, admin)
	if err != nil {
		return WebsiteDomain{}, jobs.Job{}, err
	}
	if domain.Kind == "primary" {
		return WebsiteDomain{}, jobs.Job{}, ErrPrimaryRequired
	}
	if website.Status != "active" || domain.Status == "pending" {
		return WebsiteDomain{}, jobs.Job{}, ErrWebsiteDomainBusy
	}
	kind := jobs.KindWebsiteDomainDisable
	if enabled {
		kind = jobs.KindWebsiteDomainEnable
	}
	job, err := newJob(website.NodeID, userID, kind, domainPayload{WebsiteDomainID: domain.ID})
	if err != nil {
		return WebsiteDomain{}, jobs.Job{}, err
	}
	domain, job, err = s.repository.QueueWebsiteDomainAction(ctx, domain.ID, enabled, job)
	if err == nil {
		s.notify()
	}
	return domain, job, err
}

func (s *Service) DeleteDomain(ctx context.Context, domainID, userID string, admin bool) (WebsiteDomain, jobs.Job, error) {
	website, domain, err := s.domain(ctx, domainID, userID, admin)
	if err != nil {
		return WebsiteDomain{}, jobs.Job{}, err
	}
	if domain.Kind == "primary" {
		return WebsiteDomain{}, jobs.Job{}, ErrPrimaryRequired
	}
	if website.Status != "active" || domain.Status == "pending" {
		return WebsiteDomain{}, jobs.Job{}, ErrWebsiteDomainBusy
	}
	job, err := newJob(website.NodeID, userID, jobs.KindWebsiteDomainDelete, domainPayload{WebsiteDomainID: domain.ID})
	if err != nil {
		return WebsiteDomain{}, jobs.Job{}, err
	}
	domain, job, err = s.repository.QueueWebsiteDomainAction(ctx, domain.ID, false, job)
	if err == nil {
		s.notify()
	}
	return domain, job, err
}

func (s *Service) RenameDomain(ctx context.Context, domainID, hostname, userID string, admin bool) (WebsiteDomain, jobs.Job, error) {
	website, domain, err := s.domain(ctx, domainID, userID, admin)
	if err != nil {
		return WebsiteDomain{}, jobs.Job{}, err
	}
	if website.Status != "active" || domain.Status == "pending" {
		return WebsiteDomain{}, jobs.Job{}, ErrWebsiteDomainBusy
	}
	hostname, err = validate.Domain(hostname)
	if err != nil {
		return WebsiteDomain{}, jobs.Job{}, err
	}
	if hostname == domain.Hostname {
		return WebsiteDomain{}, jobs.Job{}, ErrHostnameSame
	}
	job, err := newJob(website.NodeID, userID, jobs.KindWebsiteDomainUpdate, renamePayload{WebsiteDomainID: domain.ID, PreviousHostname: domain.Hostname, Hostname: hostname})
	if err != nil {
		return WebsiteDomain{}, jobs.Job{}, err
	}
	domain, job, err = s.repository.QueueWebsiteDomainRename(ctx, domain.ID, hostname, job)
	if err == nil {
		s.notify()
	}
	return domain, job, err
}

func (s *Service) Provision(ctx context.Context, job jobs.Job) error {
	var payload websitePayload
	if err := json.Unmarshal([]byte(job.Payload), &payload); err != nil || payload.WebsiteID == "" {
		return fmt.Errorf("decode website job payload")
	}
	website, err := s.repository.Website(ctx, payload.WebsiteID)
	if err != nil {
		return fmt.Errorf("get website: %w", err)
	}
	primary, err := s.repository.PrimaryDomain(ctx, website.ID)
	if err != nil {
		return fmt.Errorf("get primary domain: %w", err)
	}
	if err := s.ensure(ctx, website); err != nil {
		_ = s.repository.UpdateWebsiteStatus(ctx, website.ID, "error")
		_ = s.repository.UpdateWebsiteDomainStatus(ctx, primary.ID, "error")
		return fmt.Errorf("ensure website: %w", err)
	}
	if err := s.repository.UpdateWebsiteStatus(ctx, website.ID, "active"); err != nil {
		return err
	}
	return s.repository.UpdateWebsiteDomainStatus(ctx, primary.ID, "active")
}

func (s *Service) ProvisionWebsiteAction(ctx context.Context, job jobs.Job) error {
	if job.Kind != jobs.KindWebsiteEnable && job.Kind != jobs.KindWebsiteDisable && job.Kind != jobs.KindWebsiteDelete {
		return fmt.Errorf("unsupported website action: %s", job.Kind)
	}
	var payload websitePayload
	if err := json.Unmarshal([]byte(job.Payload), &payload); err != nil || payload.WebsiteID == "" {
		return fmt.Errorf("decode website action payload")
	}
	website, err := s.repository.Website(ctx, payload.WebsiteID)
	if err != nil {
		if job.Kind == jobs.KindWebsiteDelete && errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	}
	spec, node, err := s.spec(ctx, website, job.Kind == jobs.KindWebsiteEnable)
	if err != nil {
		return err
	}
	switch job.Kind {
	case jobs.KindWebsiteEnable:
		err = s.agent.EnsureWebsite(ctx, node.Endpoint, spec)
	case jobs.KindWebsiteDisable:
		err = s.agent.DisableWebsite(ctx, node.Endpoint, spec)
	case jobs.KindWebsiteDelete:
		err = s.agent.DeleteWebsite(ctx, node.Endpoint, spec)
	}
	if err != nil {
		_ = s.repository.UpdateWebsiteStatus(ctx, website.ID, "error")
		return fmt.Errorf("reconcile website action: %w", err)
	}
	if job.Kind == jobs.KindWebsiteDelete {
		return s.repository.DeleteWebsite(ctx, website.ID)
	}
	status := "disabled"
	if job.Kind == jobs.KindWebsiteEnable {
		status = "active"
	}
	return s.repository.UpdateWebsiteStatus(ctx, website.ID, status)
}

func (s *Service) ProvisionDomain(ctx context.Context, job jobs.Job) error {
	if job.Kind != jobs.KindWebsiteDomainCreate && job.Kind != jobs.KindWebsiteDomainEnable && job.Kind != jobs.KindWebsiteDomainDisable && job.Kind != jobs.KindWebsiteDomainDelete {
		return fmt.Errorf("unsupported website domain action: %s", job.Kind)
	}
	var payload domainPayload
	if err := json.Unmarshal([]byte(job.Payload), &payload); err != nil || payload.WebsiteDomainID == "" {
		return fmt.Errorf("decode website domain payload")
	}
	domain, err := s.repository.WebsiteDomain(ctx, payload.WebsiteDomainID)
	if err != nil {
		if job.Kind == jobs.KindWebsiteDomainDelete && errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	}
	website, err := s.repository.Website(ctx, domain.WebsiteID)
	if err != nil {
		return err
	}
	if err := s.ensure(ctx, website); err != nil {
		_ = s.repository.UpdateWebsiteDomainStatus(ctx, domain.ID, "error")
		return fmt.Errorf("reconcile website domain: %w", err)
	}
	if job.Kind == jobs.KindWebsiteDomainDelete {
		return s.repository.DeleteWebsiteDomain(ctx, domain.ID)
	}
	status := "active"
	if job.Kind == jobs.KindWebsiteDomainDisable {
		status = "disabled"
	}
	return s.repository.UpdateWebsiteDomainStatus(ctx, domain.ID, status)
}

func (s *Service) ProvisionDomainRename(ctx context.Context, job jobs.Job) error {
	if job.Kind != jobs.KindWebsiteDomainUpdate {
		return fmt.Errorf("unsupported website domain rename: %s", job.Kind)
	}
	var payload renamePayload
	if err := json.Unmarshal([]byte(job.Payload), &payload); err != nil || payload.WebsiteDomainID == "" || payload.PreviousHostname == "" || payload.Hostname == "" {
		return fmt.Errorf("decode website domain rename payload")
	}
	domain, err := s.repository.WebsiteDomain(ctx, payload.WebsiteDomainID)
	if err != nil {
		return err
	}
	if domain.PreviousHostname == "" && domain.Hostname == payload.Hostname && (domain.Status == "active" || domain.Status == "disabled") {
		return nil
	}
	if domain.PreviousHostname != payload.PreviousHostname || domain.Hostname != payload.Hostname {
		return fmt.Errorf("website domain rename state does not match job")
	}
	website, err := s.repository.Website(ctx, domain.WebsiteID)
	if err == nil {
		err = s.ensure(ctx, website)
	}
	if err != nil {
		if job.Attempts >= job.MaxAttempts {
			_ = s.repository.FailWebsiteDomainRename(ctx, domain.ID)
		}
		return fmt.Errorf("rename website domain: %w", err)
	}
	return s.repository.CompleteWebsiteDomainRename(ctx, domain.ID)
}

func (s *Service) website(ctx context.Context, id, userID string, admin bool) (Website, error) {
	if err := validate.ID("websiteId", id); err != nil {
		return Website{}, err
	}
	website, err := s.repository.Website(ctx, id)
	if err != nil {
		return Website{}, err
	}
	if _, err := s.accounts.Account(ctx, website.AccountID, userID, admin); err != nil {
		return Website{}, err
	}
	return website, nil
}

func (s *Service) domain(ctx context.Context, id, userID string, admin bool) (Website, WebsiteDomain, error) {
	if err := validate.ID("websiteDomainId", id); err != nil {
		return Website{}, WebsiteDomain{}, err
	}
	domain, err := s.repository.WebsiteDomain(ctx, id)
	if err != nil {
		return Website{}, WebsiteDomain{}, err
	}
	website, err := s.website(ctx, domain.WebsiteID, userID, admin)
	if err != nil {
		return Website{}, WebsiteDomain{}, err
	}
	return website, domain, nil
}

func (s *Service) ensure(ctx context.Context, website Website) error {
	spec, node, err := s.spec(ctx, website, true)
	if err != nil {
		return err
	}
	return s.agent.EnsureWebsite(ctx, node.Endpoint, spec)
}

func (s *Service) spec(ctx context.Context, website Website, activeAccount bool) (Spec, nodes.Node, error) {
	account, err := s.accounts.Get(ctx, website.AccountID)
	if err != nil {
		return Spec{}, nodes.Node{}, fmt.Errorf("get website account: %w", err)
	}
	if activeAccount && (account.Status != "active" || !account.Enabled) {
		return Spec{}, nodes.Node{}, accounts.ErrBusy
	}
	node, err := s.nodes.Node(ctx, website.NodeID)
	if err != nil {
		return Spec{}, nodes.Node{}, fmt.Errorf("get website node: %w", err)
	}
	domains, err := s.repository.EnabledWebsiteDomains(ctx, website.ID)
	if err != nil {
		return Spec{}, nodes.Node{}, err
	}
	spec := Spec{AccountID: account.ID, SystemUser: account.SystemUser, WebsiteID: website.ID, DocumentRoot: website.DocumentRoot, Kind: website.Kind, WebDriver: website.WebDriver, RuntimeDriver: website.RuntimeDriver, RuntimeVersion: website.RuntimeVersion}
	for _, domain := range domains {
		if domain.Kind == "primary" {
			spec.PrimaryDomain = domain.Hostname
		} else {
			spec.Aliases = append(spec.Aliases, domain.Hostname)
		}
	}
	if spec.PrimaryDomain == "" {
		return Spec{}, nodes.Node{}, ErrPrimaryRequired
	}
	return spec, node, nil
}

func newJob(nodeID, userID, kind string, payload any) (jobs.Job, error) {
	id, err := idgen.ID()
	if err != nil {
		return jobs.Job{}, err
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return jobs.Job{}, err
	}
	return jobs.Job{ID: id, NodeID: nodeID, UserID: userID, Kind: kind, Status: "queued", Payload: string(data), MaxAttempts: 2, CreatedAt: time.Now().UTC()}, nil
}
