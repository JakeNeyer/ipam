package handlers

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strconv"

	"github.com/JakeNeyer/ipam/network"
	"github.com/JakeNeyer/ipam/server/auth"
	"github.com/JakeNeyer/ipam/store"
	"github.com/google/uuid"
	"github.com/swaggest/rest"
	"github.com/swaggest/rest/nethttp"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
)

// cidrStartEnd returns the first and last IP of a CIDR as strings, or "", "" if invalid.
func cidrStartEnd(cidr string) (start, end string) {
	_, n, err := net.ParseCIDR(cidr)
	if err != nil {
		return "", ""
	}
	first := make(net.IP, len(n.IP))
	copy(first, n.IP)
	last := make(net.IP, len(n.IP))
	copy(last, n.IP)
	for i := range last {
		last[i] = last[i] | (n.Mask[i] ^ 0xff)
	}
	return first.String(), last.String()
}

// exportCSVOutput is the output for GET /api/export/csv. Body is written as text/csv by the CSV response encoder.
type exportCSVOutput struct {
	Body []byte
}

// NewExportCSVUseCase returns a use case for GET /api/export/csv.
func NewExportCSVUseCase(s store.Storer) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input struct{}, output *exportCSVOutput) error {
		user := auth.UserFromContext(ctx)
		orgID := auth.ResolveOrgID(ctx, user, uuid.Nil)

		var envs []*network.Environment
		var blocks []*network.Block
		var err error
		if orgID != nil {
			envs, _, err = s.ListEnvironmentsFiltered("", orgID, 0, 0)
			if err != nil {
				return status.Wrap(err, status.Internal)
			}
			blocks, _, err = s.ListBlocksFiltered("", nil, nil, orgID, false, "", nil, 0, 0)
		} else {
			envs, err = s.ListEnvironments()
			if err != nil {
				return status.Wrap(err, status.Internal)
			}
			blocks, err = s.ListBlocks()
		}
		if err != nil {
			return status.Wrap(err, status.Internal)
		}

		envByID := make(map[string]string)
		for _, e := range envs {
			envByID[e.Id.String()] = e.Name
		}

		var buf bytes.Buffer
		wr := csv.NewWriter(&buf)
		if err := wr.Write([]string{"name", "cidr", "cidr_start", "cidr_end", "environment_name", "total_ips", "used_ips", "available_ips"}); err != nil {
			return status.Wrap(err, status.Internal)
		}

		for _, b := range blocks {
			totalStr, usedStr, availStr, _ := derivedBlockUsage(s, b.Name, b.CIDR, orgID)
			envName := envByID[b.EnvironmentID.String()]
			start, end := cidrStartEnd(b.CIDR)
			_ = wr.Write([]string{
				b.Name,
				b.CIDR,
				start,
				end,
				envName,
				totalStr,
				usedStr,
				availStr,
			})
		}

		wr.Flush()
		if wr.Error() != nil {
			return status.Wrap(errors.New("failed to write CSV"), status.Internal)
		}

		output.Body = buf.Bytes()
		return nil
	})
	u.SetTitle("Export CSV")
	u.SetDescription("Exports all blocks as CSV")
	u.SetExpectedErrors(status.Internal)
	return u
}

// csvResponseEncoder writes exportCSVOutput as text/csv.
type csvResponseEncoder struct{}

// NewCSVResponseEncoder returns a ResponseEncoder that writes exportCSVOutput.Body as text/csv.
func NewCSVResponseEncoder() nethttp.ResponseEncoder {
	return &csvResponseEncoder{}
}

func (e *csvResponseEncoder) WriteErrResponse(w http.ResponseWriter, _ *http.Request, statusCode int, response interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(response)
}

func (e *csvResponseEncoder) WriteSuccessfulResponse(w http.ResponseWriter, _ *http.Request, output interface{}, _ rest.HandlerTrait) {
	out, ok := output.(*exportCSVOutput)
	if !ok || out == nil {
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="ipam-export.csv"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(out.Body)
}

func (e *csvResponseEncoder) SetupOutput(output interface{}, _ *rest.HandlerTrait) {}

func (e *csvResponseEncoder) MakeOutput(w http.ResponseWriter, _ rest.HandlerTrait) interface{} {
	return &exportCSVOutput{}
}

// ExportCSVHandler returns an http.Handler that writes CSV directly to the response.
// Use this when the use-case + custom encoder chain does not write the body (e.g. empty download).
func ExportCSVHandler(s store.Storer) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		ctx := r.Context()
		user := auth.UserFromContext(ctx)
		orgID := auth.ResolveOrgID(ctx, user, uuid.Nil)

		var envs []*network.Environment
		var blocks []*network.Block
		var err error
		if orgID != nil {
			envs, _, err = s.ListEnvironmentsFiltered("", orgID, 0, 0)
			if err != nil {
				http.Error(w, "Failed to list environments", http.StatusInternalServerError)
				return
			}
			blocks, _, err = s.ListBlocksFiltered("", nil, nil, orgID, false, "", nil, 0, 0)
		} else {
			envs, err = s.ListEnvironments()
			if err != nil {
				http.Error(w, "Failed to list environments", http.StatusInternalServerError)
				return
			}
			blocks, err = s.ListBlocks()
		}
		if err != nil {
			http.Error(w, "Failed to list blocks", http.StatusInternalServerError)
			return
		}
		envByID := make(map[string]string)
		for _, e := range envs {
			envByID[e.Id.String()] = e.Name
		}
		var buf bytes.Buffer
		wr := csv.NewWriter(&buf)
		_ = wr.Write([]string{"name", "cidr", "cidr_start", "cidr_end", "environment_name", "total_ips", "used_ips", "available_ips"})
		for _, b := range blocks {
			totalStr, usedStr, availStr, _ := derivedBlockUsage(s, b.Name, b.CIDR, orgID)
			envName := envByID[b.EnvironmentID.String()]
			start, end := cidrStartEnd(b.CIDR)
			_ = wr.Write([]string{
				b.Name,
				b.CIDR,
				start,
				end,
				envName,
				totalStr,
				usedStr,
				availStr,
			})
		}
		wr.Flush()
		if wr.Error() != nil {
			http.Error(w, "Failed to write CSV", http.StatusInternalServerError)
			return
		}
		body := buf.Bytes()
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="ipam-export.csv"`)
		w.Header().Set("Content-Length", strconv.Itoa(len(body)))
		w.WriteHeader(http.StatusOK)
		if r.Method != http.MethodHead {
			_, _ = w.Write(body)
		}
	})
}
