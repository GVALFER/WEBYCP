package sqlite

import (
	"context"
	"time"

	"github.com/GVALFER/WEBYCP/internal/certificates"
	"github.com/GVALFER/WEBYCP/internal/jobs"
	"github.com/GVALFER/WEBYCP/internal/pagination"
	"github.com/GVALFER/WEBYCP/internal/store/sqlite/dbgen"
)

func (s *Store) Certificates(ctx context.Context, userID string, admin bool) ([]certificates.Certificate, error) {
	rows, err := s.queries.ListCertificates(ctx, dbgen.ListCertificatesParams{Kind: "", IsAdmin: admin, UserID: userID})
	if err != nil {
		return nil, err
	}
	result := make([]certificates.Certificate, 0, len(rows))
	for _, row := range rows {
		value, err := s.certificateValue(ctx, row)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}

func (s *Store) CertificatePage(
	ctx context.Context, userID string, admin bool, kind string, query pagination.Query,
) (pagination.Result[certificates.Certificate], error) {
	total, err := s.queries.CountCertificates(ctx, dbgen.CountCertificatesParams{
		Kind: kind, IsAdmin: admin, UserID: userID,
	})
	if err != nil {
		return pagination.Result[certificates.Certificate]{}, err
	}
	query = pagination.Clamp(query, total)
	rows, err := s.queries.ListCertificatesPage(ctx, dbgen.ListCertificatesPageParams{
		Kind: kind, IsAdmin: admin, UserID: userID,
		PageOffset: pagination.Offset(query), PageSize: int64(query.Size),
	})
	if err != nil {
		return pagination.Result[certificates.Certificate]{}, err
	}
	items := make([]certificates.Certificate, 0, len(rows))
	for _, row := range rows {
		value, valueErr := s.certificateValue(ctx, row)
		if valueErr != nil {
			return pagination.Result[certificates.Certificate]{}, valueErr
		}
		items = append(items, value)
	}
	return pagination.Result[certificates.Certificate]{Items: items, Query: query, Total: total}, nil
}

func (s *Store) Certificate(ctx context.Context, id string) (certificates.Certificate, error) {
	row, err := s.queries.GetCertificate(ctx, id)
	if err != nil {
		return certificates.Certificate{}, err
	}
	return s.certificateValue(ctx, row)
}

func (s *Store) WebsiteCertificate(ctx context.Context, websiteID string) (certificates.Certificate, error) {
	row, err := s.queries.GetWebsiteCertificate(ctx, nullString(websiteID))
	if err != nil {
		return certificates.Certificate{}, err
	}
	return s.certificateValue(ctx, row)
}

func (s *Store) PanelCertificate(ctx context.Context) (certificates.Certificate, error) {
	row, err := s.queries.GetPanelCertificate(ctx)
	if err != nil {
		return certificates.Certificate{}, err
	}
	return s.certificateValue(ctx, row)
}

func (s *Store) QueueCertificate(ctx context.Context, value certificates.Certificate, job jobs.Job, create bool) (certificates.Certificate, jobs.Job, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return certificates.Certificate{}, jobs.Job{}, err
	}
	defer tx.Rollback()
	q := s.queries.WithTx(tx)
	var row dbgen.Certificate
	if create {
		row, err = q.CreateCertificate(ctx, dbgen.CreateCertificateParams{
			ID: value.ID, WebsiteID: nullString(value.WebsiteID), NodeID: value.NodeID,
			Kind: value.Kind, Name: value.Name, Email: value.Email,
			RedirectHttps: boolValue(value.RedirectHTTPS), CreatedAt: timeValue(value.CreatedAt), UpdatedAt: timeValue(time.Now().UTC()),
		})
	} else {
		err = q.QueueCertificate(ctx, dbgen.QueueCertificateParams{Name: value.Name, Email: value.Email, RedirectHttps: boolValue(value.RedirectHTTPS), UpdatedAt: timeValue(time.Now().UTC()), ID: value.ID})
		if err == nil {
			row, err = q.GetCertificate(ctx, value.ID)
		}
	}
	if err != nil {
		return certificates.Certificate{}, jobs.Job{}, err
	}
	if err := replaceCertificateNames(ctx, q, value.ID, value.Names); err != nil {
		return certificates.Certificate{}, jobs.Job{}, err
	}
	created, err := createJob(ctx, q, job)
	if err != nil {
		return certificates.Certificate{}, jobs.Job{}, err
	}
	if err := tx.Commit(); err != nil {
		return certificates.Certificate{}, jobs.Job{}, err
	}
	result := certificateRowValue(row)
	result.Names = append([]string(nil), value.Names...)
	return result, created, nil
}

func (s *Store) SetResult(ctx context.Context, id string, names []string, status string, expiresAt, renewAfter *time.Time, message string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	q := s.queries.WithTx(tx)
	if err := q.SetCertificateResult(ctx, dbgen.SetCertificateResultParams{
		Status: status, ExpiresAt: nullTime(expiresAt), RenewAfter: nullTime(renewAfter), Error: message,
		UpdatedAt: timeValue(time.Now().UTC()), ID: id,
	}); err != nil {
		return err
	}
	if err := replaceCertificateNames(ctx, q, id, names); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) DueCertificates(ctx context.Context, now time.Time) ([]certificates.Certificate, error) {
	rows, err := s.queries.ListDueCertificates(ctx, nullTime(&now))
	if err != nil {
		return nil, err
	}
	result := make([]certificates.Certificate, 0, len(rows))
	for _, row := range rows {
		value, err := s.certificateValue(ctx, row)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}

func (s *Store) CertificateJobPending(ctx context.Context, id string) (bool, error) {
	return s.queries.CertificatePendingJobExists(ctx, id)
}

func replaceCertificateNames(ctx context.Context, q *dbgen.Queries, id string, names []string) error {
	if err := q.DeleteCertificateNames(ctx, id); err != nil {
		return err
	}
	for _, name := range names {
		if err := q.ReplaceCertificateName(ctx, dbgen.ReplaceCertificateNameParams{CertificateID: id, Name: name}); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) certificateValue(ctx context.Context, row dbgen.Certificate) (certificates.Certificate, error) {
	value := certificateRowValue(row)
	names, err := s.queries.ListCertificateNames(ctx, row.ID)
	if err != nil {
		return certificates.Certificate{}, err
	}
	value.Names = names
	return value, nil
}

func certificateRowValue(row dbgen.Certificate) certificates.Certificate {
	return certificates.Certificate{
		ID: row.ID, WebsiteID: row.WebsiteID.String, NodeID: row.NodeID, Kind: row.Kind,
		Name: row.Name, Email: row.Email, Status: row.Status, RedirectHTTPS: row.RedirectHttps != 0,
		ExpiresAt: timePtr(row.ExpiresAt), RenewAfter: timePtr(row.RenewAfter), Error: row.Error,
		CreatedAt: timeFrom(row.CreatedAt), UpdatedAt: timeFrom(row.UpdatedAt),
	}
}
